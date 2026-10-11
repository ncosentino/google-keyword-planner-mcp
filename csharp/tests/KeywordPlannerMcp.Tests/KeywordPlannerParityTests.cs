using System.Globalization;
using System.Net;
using System.Text.Json;
using KeywordPlannerMcp.KeywordPlanner;
using KeywordPlannerMcp.Tools;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Xunit;

namespace KeywordPlannerMcp.Tests;

public sealed class KeywordPlannerParityTests
{
    private const string ForecastBody = """{"campaignForecastMetrics":{"clicks":12.5,"costMicros":"9000000000","averageCpcMicros":"720000"}}""";
    private const string HistoricalBody = """
        {"results":[{"text":"dotnet","keywordMetrics":{"avgMonthlySearches":"5000",
        "competition":"LOW","competitionIndex":"23","lowTopOfPageBidMicros":"200000",
        "highTopOfPageBidMicros":"800000","monthlySearchVolumes":[
        {"year":"2026","month":"SEPTEMBER","monthlySearches":"4500"}]}}]}
        """;

    private sealed class RecordingPacer : IPlanningPacer
    {
        public int Calls { get; private set; }
        public bool Cancel { get; set; }
        public Task WaitAsync(CancellationToken cancellationToken)
        {
            cancellationToken.ThrowIfCancellationRequested();
            Calls++;
            return Cancel ? Task.FromCanceled(new CancellationToken(true)) : Task.CompletedTask;
        }
    }

    private sealed class Handler : HttpMessageHandler
    {
        public List<string> Bodies { get; } = [];
        public List<string> Endpoints { get; } = [];
        public int TokenCalls { get; private set; }
        public string? ForecastResponse { get; set; }
        public Func<int, string>? ForecastResponder { get; set; }

        protected override async Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            string body;
            if (request.RequestUri!.Host == "oauth2.googleapis.com")
            {
                TokenCalls++;
                body = """{"access_token":"fake","expires_in":3600}""";
            }
            else
            {
                Endpoints.Add(request.RequestUri.AbsoluteUri);
                Bodies.Add(await request.Content!.ReadAsStringAsync(cancellationToken));
                body = request.RequestUri.AbsoluteUri.EndsWith("generateKeywordForecastMetrics", StringComparison.Ordinal)
                    ? ForecastResponder?.Invoke(Bodies.Count) ?? ForecastResponse ?? ForecastBody
                    : request.RequestUri.AbsoluteUri.EndsWith("generateKeywordHistoricalMetrics", StringComparison.Ordinal)
                        ? HistoricalBody : """{"results":[]}""";
            }
            return new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent(body) };
        }
    }

    private static KeywordPlannerClient Client(Handler handler, RecordingPacer pacer,
        string? language = null, IReadOnlyList<string>? geography = null, TimeProvider? clock = null)
    {
        var http = new HttpClient(handler);
        return new KeywordPlannerClient("dev", "123", null,
            new OAuth2TokenProvider("id", "secret", "refresh", http), http,
            language, geography, clock, pacer);
    }

    [Fact]
    public async Task HistoricalMetrics_DecodesRealRestShapeAndInt64Strings()
    {
        var handler = new Handler();
        var result = await Client(handler, new RecordingPacer()).GetHistoricalMetricsAsync(["dotnet"]);
        Assert.Equal(1, result.Count);
        var keyword = Assert.Single(result.Keywords);
        Assert.Equal(5000L, keyword.AvgMonthlySearches);
        Assert.Equal(23, keyword.CompetitionIndex);
        Assert.Equal(200000L, keyword.LowTopOfPageBidMicros);
        Assert.Equal(new MonthlyVolume(2026, 9, 4500), Assert.Single(keyword.MonthlySearchVolumes));
        Assert.Contains("/v25/customers/123:", Assert.Single(handler.Endpoints), StringComparison.Ordinal);
    }

    [Theory]
    [InlineData(0, 30, "2027-01-31")]
    [InlineData(-1, 30, "2027-01-31")]
    [InlineData(1, 1, "2027-01-02")]
    [InlineData(364, 364, "2027-12-31")]
    public async Task Forecast_UsesV25ShapeInclusiveUtcCalendarDatesAndDefaults(
        int days, int expectedDays, string endDate)
    {
        var handler = new Handler();
        var clock = new ManualClock(new DateTimeOffset(2026, 12, 31, 23, 59, 59, TimeSpan.Zero));
        var result = await Client(handler, new RecordingPacer(), clock: clock).GetKeywordForecastAsync(
            ["dotnet", "csharp"], 0, days, language: "languageConstants/1000",
            geoTargetConstants: ["geoTargetConstants/2840"], matchType: "exact", currencyCode: "USD");
        Assert.Empty(result.Keywords);
        Assert.Equal(12.5, result.Total.Clicks);
        Assert.Equal(9_000_000_000L, result.Total.CostMicros);
        Assert.Equal(720000L, result.Total.AverageCpcMicros);
        Assert.Equal(expectedDays, result.ForecastDays);
        Assert.Equal(1_000_000L, result.MaxCpcMicros);
        Assert.Equal("2027-01-02", result.StartDate);
        Assert.Equal(endDate, result.EndDate);
        Assert.Equal("USD", result.CurrencyCode);
        Assert.Equal("EXACT", result.MatchType);
        Assert.Equal("languageConstants/1000", result.Language);
        using var json = JsonDocument.Parse(Assert.Single(handler.Bodies));
        var root = json.RootElement;
        Assert.Equal("USD", root.GetProperty("currencyCode").GetString());
        Assert.Equal(endDate, root.GetProperty("forecastPeriod").GetProperty("endDate").GetString());
        Assert.False(root.TryGetProperty("campaignForecastSpec", out _));
        var campaign = root.GetProperty("campaign");
        Assert.Equal("languageConstants/1000", campaign.GetProperty("languageConstants")[0].GetString());
        Assert.Equal("geoTargetConstants/2840", campaign.GetProperty("geoTargetConstants")[0].GetString());
        Assert.Equal("1000000", campaign.GetProperty("biddingStrategy")
            .GetProperty("manualCpcBiddingStrategy").GetProperty("maxCpcBidMicros").GetString());
        var group = campaign.GetProperty("adGroups")[0];
        Assert.False(group.TryGetProperty("biddableKeywords", out _));
        Assert.Equal("dotnet", group.GetProperty("keywords")[0].GetProperty("text").GetString());
        Assert.Equal("EXACT", group.GetProperty("keywords")[0].GetProperty("matchType").GetString());
    }

    [Theory]
    [InlineData(false, 1, 0)]
    [InlineData(true, 3, 2)]
    public async Task Forecast_BreakdownIsOptInIndependentAndNotSummed(bool breakdown, int calls, int scenarios)
    {
        var handler = new Handler
        {
            ForecastResponder = call => """{"campaignForecastMetrics":{"clicks":"""
                + (call == 3 ? 25 : call * 10).ToString(CultureInfo.InvariantCulture) + "}}"
        };
        var pacer = new RecordingPacer();
        var result = await Client(handler, pacer).GetKeywordForecastAsync(
            ["a", "b"], 1_000_000, 30, includeKeywordBreakdown: breakdown);
        Assert.Equal(calls, handler.Bodies.Count);
        Assert.Equal(calls, pacer.Calls);
        Assert.Equal(scenarios, result.Keywords.Count);
        Assert.Equal(calls == 3 ? 25d : 10d, result.Total.Clicks);
        Assert.Null(result.Total.Text);
        if (breakdown)
        {
            Assert.Equal("a", result.Keywords[0].Text);
            Assert.Equal("b", result.Keywords[1].Text);
            Assert.NotEqual(result.Keywords.Sum(s => s.Clicks), result.Total.Clicks);
            using var first = JsonDocument.Parse(handler.Bodies[0]);
            Assert.Equal(1, first.RootElement.GetProperty("campaign").GetProperty("adGroups")[0]
                .GetProperty("keywords").GetArrayLength());
            using var combined = JsonDocument.Parse(handler.Bodies[2]);
            Assert.Equal(2, combined.RootElement.GetProperty("campaign").GetProperty("adGroups")[0]
                .GetProperty("keywords").GetArrayLength());
        }
    }

    [Fact]
    public async Task Forecast_SingleKeywordBreakdownReusesItsResult()
    {
        var handler = new Handler();
        var result = await Client(handler, new RecordingPacer()).GetKeywordForecastAsync(
            ["a"], 1, 30, includeKeywordBreakdown: true);
        Assert.Single(handler.Bodies);
        Assert.Equal(Assert.Single(result.Keywords) with { Text = null }, result.Total);
    }

    [Fact]
    public async Task Forecast_MissingAggregateIsAnExplicitError()
    {
        var handler = new Handler { ForecastResponse = "{}" };
        var exception = await Assert.ThrowsAsync<InvalidOperationException>(() =>
            Client(handler, new RecordingPacer()).GetKeywordForecastAsync(["a"], 1, 30));
        Assert.Contains("campaignForecastMetrics", exception.Message, StringComparison.Ordinal);
    }

    [Fact]
    public async Task Forecast_UnrestrictedDefaultsOmitTargetingAndCurrencyAndNormalizeNegativeBid()
    {
        var handler = new Handler();
        var result = await Client(handler, new RecordingPacer()).GetKeywordForecastAsync(["a"], -1, 0);
        Assert.Equal(1_000_000L, result.MaxCpcMicros);
        Assert.Equal(30, result.ForecastDays);
        Assert.Equal("BROAD", result.MatchType);
        Assert.Null(result.CurrencyCode);
        Assert.Null(result.Language);
        Assert.Null(result.GeoTargetConstants);
        using var json = JsonDocument.Parse(Assert.Single(handler.Bodies));
        Assert.False(json.RootElement.TryGetProperty("currencyCode", out _));
        var campaign = json.RootElement.GetProperty("campaign");
        Assert.False(campaign.TryGetProperty("languageConstants", out _));
        Assert.False(campaign.TryGetProperty("geoTargetConstants", out _));
    }

    [Theory]
    [InlineData("{}", false)]
    [InlineData("""{"clicks":0,"costMicros":"0","averageCpcMicros":"0"}""", true)]
    public async Task Forecast_AbsentMetricsAreOmittedButZerosArePresent(string metrics, bool present)
    {
        var handler = new Handler { ForecastResponse = """{"campaignForecastMetrics":""" + metrics + "}" };
        var result = await Client(handler, new RecordingPacer()).GetKeywordForecastAsync(["a"], 1, 30);
        using var json = JsonDocument.Parse(JsonSerializer.Serialize(result, KwpJsonContext.Default.ForecastResponse));
        var total = json.RootElement.GetProperty("total");
        foreach (var field in new[] { "clicks", "costMicros", "averageCpcMicros" })
        {
            Assert.Equal(present, total.TryGetProperty(field, out var value));
            if (present) Assert.Equal(0d, value.GetDouble());
        }
        foreach (var field in new[] { "impressions", "ctr", "conversions", "averageCpaMicros" })
            Assert.False(total.TryGetProperty(field, out _));
    }

    [Fact]
    public async Task Targeting_DefaultsOverridesAndNetworkAreAppliedAcrossEndpoints()
    {
        var handler = new Handler();
        var pacer = new RecordingPacer();
        var client = Client(handler, pacer, "languageConstants/1000",
            ["geoTargetConstants/2840", " geoTargetConstants/2840 "]);
        await client.GenerateKeywordIdeasAsync(["a"], null, "", keywordPlanNetwork: "GOOGLE_SEARCH");
        await client.GetHistoricalMetricsAsync(["a"]);
        await client.GetKeywordForecastAsync(["a"], 1, 1);
        for (var i = 0; i < 3; i++)
        {
            using var json = JsonDocument.Parse(handler.Bodies[i]);
            var targeting = i == 2 ? json.RootElement.GetProperty("campaign") : json.RootElement;
            Assert.Equal(1, targeting.GetProperty("geoTargetConstants").GetArrayLength());
            Assert.Equal("languageConstants/1000", i == 2
                ? targeting.GetProperty("languageConstants")[0].GetString()
                : targeting.GetProperty("language").GetString());
        }
        using (var ideas = JsonDocument.Parse(handler.Bodies[0]))
            Assert.Equal("GOOGLE_SEARCH", ideas.RootElement.GetProperty("keywordPlanNetwork").GetString());
        await client.GetHistoricalMetricsAsync(["a"], language: "all", geoTargetConstants: [],
            keywordPlanNetwork: "GOOGLE_SEARCH_AND_PARTNERS");
        using (var unrestricted = JsonDocument.Parse(handler.Bodies[3]))
        {
            Assert.False(unrestricted.RootElement.TryGetProperty("language", out _));
            Assert.False(unrestricted.RootElement.TryGetProperty("geoTargetConstants", out _));
            Assert.Equal("GOOGLE_SEARCH_AND_PARTNERS",
                unrestricted.RootElement.GetProperty("keywordPlanNetwork").GetString());
        }
        await client.GetKeywordForecastAsync(["a"], 1, 1, language: "languageConstants/1001",
            geoTargetConstants: ["geoTargetConstants/2036"]);
        using (var overridden = JsonDocument.Parse(handler.Bodies[4]))
        {
            var campaign = overridden.RootElement.GetProperty("campaign");
            Assert.Equal("languageConstants/1001", campaign.GetProperty("languageConstants")[0].GetString());
            Assert.Equal("geoTargetConstants/2036", campaign.GetProperty("geoTargetConstants")[0].GetString());
        }
        Assert.Equal(5, pacer.Calls);
        Assert.Equal(1, handler.TokenCalls);
    }

    [Fact]
    public async Task Targeting_RejectsBlankEntriesExcessTargetsAndInvalidEnumsBeforeHttp()
    {
        var handler = new Handler();
        var client = Client(handler, new RecordingPacer());
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetHistoricalMetricsAsync(["a"], language: " "));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GenerateKeywordIdeasAsync(["a"], null, " "));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetHistoricalMetricsAsync(["a"], geoTargetConstants: [""]));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetHistoricalMetricsAsync(["a"],
            geoTargetConstants: Enumerable.Range(1, 11).Select(n => $"geoTargetConstants/{n}").ToArray()));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetHistoricalMetricsAsync(["a"], keywordPlanNetwork: "other"));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetKeywordForecastAsync(["a"], 1, 365));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetKeywordForecastAsync(["a"], 1, 1, matchType: "other"));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetKeywordForecastAsync(["a"], 1, 1, matchType: " "));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetKeywordForecastAsync([], 1, 1));
        Assert.Empty(handler.Bodies);
        Assert.Equal(0, handler.TokenCalls);
    }

    [Fact]
    public async Task Targeting_AllowsTenUniqueTargetsAndDeduplicatesBeforeLimit()
    {
        var handler = new Handler();
        var targets = Enumerable.Range(1, 10).Select(n => $"geoTargetConstants/{n}").ToArray();
        await Client(handler, new RecordingPacer()).GetHistoricalMetricsAsync(["a"],
            geoTargetConstants: targets.Concat(targets).ToArray());
        using var json = JsonDocument.Parse(Assert.Single(handler.Bodies));
        Assert.Equal(10, json.RootElement.GetProperty("geoTargetConstants").GetArrayLength());
    }

    [Fact]
    public async Task Tools_PreserveCommaSeparatedTargetingAndExplicitWorldwideOverride()
    {
        var handler = new Handler();
        var tool = new KeywordPlannerTool(Client(handler, new RecordingPacer(),
            "languageConstants/1000", ["geoTargetConstants/2840"]));
        await tool.GetHistoricalMetricsAsync("a,b", language: "all", geoTargetConstants: "");
        using (var json = JsonDocument.Parse(handler.Bodies[0]))
        {
            Assert.Equal(2, json.RootElement.GetProperty("keywords").GetArrayLength());
            Assert.False(json.RootElement.TryGetProperty("language", out _));
            Assert.False(json.RootElement.TryGetProperty("geoTargetConstants", out _));
        }
        var error = await tool.GetHistoricalMetricsAsync("a", geoTargetConstants: "geoTargetConstants/2840, ");
        Assert.Contains("must not be blank", error, StringComparison.Ordinal);
        Assert.Single(handler.Bodies);
    }

    [Fact]
    public async Task Hosting_UsesEnvironmentDefaultsAndOneSharedClientPerConfiguredCustomer()
    {
        var handler = new Handler();
        var builder = Host.CreateApplicationBuilder();
        Hosting.ConfigureCommonServices(builder, "dev", "id", "secret", "refresh", "123", null, handler,
            name => name switch
            {
                "GOOGLE_ADS_DEFAULT_LANGUAGE" => "languageConstants/1000",
                "GOOGLE_ADS_DEFAULT_GEO_TARGETS" => "geoTargetConstants/2840, geoTargetConstants/2840",
                _ => null
            });
        using var host = builder.Build();
        var client = host.Services.GetRequiredService<KeywordPlannerClient>();
        Assert.Same(client, host.Services.GetRequiredService<KeywordPlannerClient>());
        Assert.Same(host.Services.GetRequiredService<OAuth2TokenProvider>(),
            host.Services.GetRequiredService<OAuth2TokenProvider>());
        await client.GetHistoricalMetricsAsync(["a"]);
        using var json = JsonDocument.Parse(Assert.Single(handler.Bodies));
        Assert.Equal("languageConstants/1000", json.RootElement.GetProperty("language").GetString());
        Assert.Equal(1, json.RootElement.GetProperty("geoTargetConstants").GetArrayLength());
    }

    [Fact]
    public async Task Cancellation_PreventsApiRequestAndPropagatesThroughTools()
    {
        var handler = new Handler();
        var pacer = new RecordingPacer { Cancel = true };
        var client = Client(handler, pacer);
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => client.GetHistoricalMetricsAsync(["a"]));
        Assert.Empty(handler.Bodies);
        using var cancellation = new CancellationTokenSource();
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() =>
            new KeywordPlannerTool(client).GetKeywordForecastAsync("a", cancellationToken: cancellation.Token));
        Assert.Empty(handler.Bodies);
    }

    [Fact]
    public async Task PlanningPacer_RequiresOneSecondAndCancellationDoesNotConsumeNextSlot()
    {
        var clock = new ManualClock(DateTimeOffset.UnixEpoch);
        using var pacer = new PlanningPacer(clock);
        await pacer.WaitAsync(CancellationToken.None);
        using var cancellation = new CancellationTokenSource();
        var cancelled = pacer.WaitAsync(cancellation.Token);
        Assert.False(cancelled.IsCompleted);
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => cancelled);
        var next = pacer.WaitAsync(CancellationToken.None);
        Assert.False(next.IsCompleted);
        clock.Advance(TimeSpan.FromMilliseconds(999));
        Assert.False(next.IsCompleted);
        clock.Advance(TimeSpan.FromMilliseconds(1));
        await next;
    }

    [Fact]
    public async Task PlanningPacer_CancellationWhileQueuedDoesNotBlockNextRequest()
    {
        var clock = new ManualClock(DateTimeOffset.UnixEpoch);
        using var pacer = new PlanningPacer(clock);
        await pacer.WaitAsync(CancellationToken.None);
        var waiting = pacer.WaitAsync(CancellationToken.None);
        using var cancellation = new CancellationTokenSource();
        var queued = pacer.WaitAsync(cancellation.Token);
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => queued);
        clock.Advance(TimeSpan.FromSeconds(1));
        await waiting;
        var next = pacer.WaitAsync(CancellationToken.None);
        Assert.False(next.IsCompleted);
        clock.Advance(TimeSpan.FromSeconds(1));
        await next;
    }

    [Fact]
    public async Task OAuthTokenRefresh_IsSharedAcrossConcurrentRequests()
    {
        using var handler = new BlockingTokenHandler();
        using var http = new HttpClient(handler);
        using var provider = new OAuth2TokenProvider("id", "secret", "refresh", http);
        var first = provider.GetAccessTokenAsync();
        await handler.Entered.Task;
        var second = provider.GetAccessTokenAsync();
        Assert.Equal(1, handler.Calls);
        handler.Release.SetResult();
        var tokens = await Task.WhenAll(first, second);
        Assert.All(tokens, token => Assert.Equal("fake", token));
        Assert.Equal(1, handler.Calls);
    }

    [Fact]
    public async Task OAuthTokenRefresh_CancellationWhileQueuedLeavesFirstRequestUsable()
    {
        using var handler = new BlockingTokenHandler();
        using var http = new HttpClient(handler);
        using var provider = new OAuth2TokenProvider("id", "secret", "refresh", http);
        var first = provider.GetAccessTokenAsync();
        await handler.Entered.Task;
        using var cancellation = new CancellationTokenSource();
        var second = provider.GetAccessTokenAsync(cancellation.Token);
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => second);
        handler.Release.SetResult();
        Assert.Equal("fake", await first);
        Assert.Equal("fake", await provider.GetAccessTokenAsync());
        Assert.Equal(1, handler.Calls);
    }

    private sealed class BlockingTokenHandler : HttpMessageHandler
    {
        public TaskCompletionSource Entered { get; } = new(TaskCreationOptions.RunContinuationsAsynchronously);
        public TaskCompletionSource Release { get; } = new(TaskCreationOptions.RunContinuationsAsynchronously);
        public int Calls { get; private set; }
        protected override async Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            Calls++;
            Entered.TrySetResult();
            await Release.Task.WaitAsync(cancellationToken);
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent("""{"access_token":"fake","expires_in":3600}""")
            };
        }
    }

    private sealed class ManualClock(DateTimeOffset now) : TimeProvider
    {
        private readonly List<ManualTimer> _timers = [];
        private long _timestamp;
        public override long TimestampFrequency => TimeSpan.TicksPerSecond;
        public override long GetTimestamp() => _timestamp;
        public override DateTimeOffset GetUtcNow() => now.AddTicks(_timestamp);
        public override ITimer CreateTimer(TimerCallback callback, object? state, TimeSpan dueTime, TimeSpan period)
        {
            var timer = new ManualTimer(this, callback, state);
            _timers.Add(timer);
            timer.Change(dueTime, period);
            return timer;
        }
        public void Advance(TimeSpan duration)
        {
            _timestamp += duration.Ticks;
            foreach (var timer in _timers.ToArray()) timer.FireIfDue();
        }
        private sealed class ManualTimer(ManualClock clock, TimerCallback callback, object? state) : ITimer
        {
            private long? _due;
            public bool Change(TimeSpan dueTime, TimeSpan period)
            {
                _due = dueTime == Timeout.InfiniteTimeSpan ? null : clock._timestamp + dueTime.Ticks;
                return true;
            }
            public void FireIfDue()
            {
                if (_due is { } due && due <= clock._timestamp)
                {
                    _due = null;
                    callback(state);
                }
            }
            public void Dispose() => _due = null;
            public ValueTask DisposeAsync()
            {
                Dispose();
                return ValueTask.CompletedTask;
            }
        }
    }
}
