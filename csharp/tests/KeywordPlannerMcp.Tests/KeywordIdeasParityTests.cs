using System.Net;
using System.Globalization;
using System.Text.Json;
using KeywordPlannerMcp.KeywordPlanner;
using KeywordPlannerMcp.Tools;
using Xunit;

namespace KeywordPlannerMcp.Tests;

public sealed class KeywordIdeasParityTests
{
    private sealed class Pacer : IPlanningPacer
    {
        private int _calls;
        public int Calls => _calls;
        public Task WaitAsync(CancellationToken cancellationToken)
        {
            cancellationToken.ThrowIfCancellationRequested();
            Interlocked.Increment(ref _calls);
            return Task.CompletedTask;
        }
    }

    private sealed class Handler : HttpMessageHandler
    {
        private readonly object _gate = new();
        public List<(string Address, string Body)> Requests { get; } = [];
        public Func<string, string, HttpResponseMessage> Respond { get; set; } =
            (_, _) => Response("""{"results":[]}""");
        public Func<string, string, CancellationToken, Task<HttpResponseMessage>>? RespondAsync { get; set; }

        protected override async Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            if (request.RequestUri!.Host == "oauth2.googleapis.com")
                return Response("""{"access_token":"fake","expires_in":3600}""");
            var address = request.RequestUri.AbsoluteUri;
            var body = await request.Content!.ReadAsStringAsync(cancellationToken);
            lock (_gate) Requests.Add((address, body));
            return RespondAsync is null ? Respond(address, body) : await RespondAsync(address, body, cancellationToken);
        }
    }

    private static HttpResponseMessage Response(string body, HttpStatusCode status = HttpStatusCode.OK) =>
        new(status) { Content = new StringContent(body) };

    private static KeywordPlannerClient Client(HttpMessageHandler handler, Pacer pacer,
        string? language = null, IReadOnlyList<string>? geography = null)
    {
        var http = new HttpClient(handler);
        return new KeywordPlannerClient("dev", "123", null,
            new OAuth2TokenProvider("id", "secret", "refresh", http), http,
            language, geography, planningPacer: pacer);
    }

    [Fact]
    public async Task Ideas_FetchEveryPageThenGloballyRankFilterAndLimitWithSeasonality()
    {
        var handler = new Handler
        {
            Respond = (_, body) => body.Contains("pageToken", StringComparison.Ordinal)
                ? Response("""
                    {"results":[{"text":"best","keywordIdeaMetrics":{"avgMonthlySearches":"900",
                    "competitionIndex":"0","monthlySearchVolumes":[{"year":"2026","month":"JANUARY","monthlySearches":"800"}]}},
                    {"text":"tie-second","keywordIdeaMetrics":{"avgMonthlySearches":"500"}}]}
                    """)
                : Response("""
                    {"nextPageToken":"second","results":[
                    {"text":"small","keywordIdeaMetrics":{"avgMonthlySearches":"20"}},
                    {"text":"tie-first","keywordIdeaMetrics":{"avgMonthlySearches":"500"}}]}
                    """)
        };
        var pacer = new Pacer();
        var result = await Client(handler, pacer).GenerateKeywordIdeasAsync(["seed"], null, "1000",
            geoTargetConstants: ["2840"], options: new IdeaOptions(3, 100, true));
        Assert.Equal(new[] { "best", "tie-first", "tie-second" }, result.Ideas.Select(i => i.Text));
        Assert.Equal(4, result.TotalFound);
        Assert.Equal(3, result.Count);
        Assert.Equal("languageConstants/1000", result.Language);
        Assert.Equal(new[] { "geoTargetConstants/2840" }, result.GeoTargetConstants);
        Assert.Equal(new MonthlyVolume(2026, 1, 800), Assert.Single(result.Ideas[0].MonthlySearchVolumes!));
        using var json = JsonDocument.Parse(JsonSerializer.Serialize(result, KwpJsonContext.Default.KeywordIdeasResponse));
        Assert.Equal(0, json.RootElement.GetProperty("ideas")[0].GetProperty("competitionIndex").GetInt32());
        using var second = JsonDocument.Parse(handler.Requests[1].Body);
        Assert.Equal("second", second.RootElement.GetProperty("pageToken").GetString());
        Assert.Equal(2, pacer.Calls);
    }

    [Fact]
    public async Task Ideas_RepeatedPageTokenFailsInsteadOfReturningPartialResults()
    {
        var handler = new Handler { Respond = (_, _) => Response("""{"results":[],"nextPageToken":"repeat"}""") };
        var error = await Assert.ThrowsAsync<InvalidOperationException>(() =>
            Client(handler, new Pacer()).GenerateKeywordIdeasAsync(["seed"], null, null));
        Assert.Contains("pagination token", error.Message);
        Assert.Equal(2, handler.Requests.Count);
    }

    [Theory]
    [InlineData(50, 50)]
    [InlineData(0, 50)]
    [InlineData(-1, 60)]
    [InlineData(2, 2)]
    public async Task Ideas_ToolDefaultsAndClientZeroOptionsHaveDifferentLimits(int limit, int expected)
    {
        var rows = string.Join(",", Enumerable.Range(1, 60).Select(i =>
            "{\"text\":\"keyword" + i.ToString(CultureInfo.InvariantCulture)
            + "\",\"keywordIdeaMetrics\":{\"avgMonthlySearches\":\""
            + i.ToString(CultureInfo.InvariantCulture) + "\"}}"));
        var handler = new Handler { Respond = (_, _) => Response("""{"results":[""" + rows + "]}") };
        var client = Client(handler, new Pacer());
        var unlimited = await client.GenerateKeywordIdeasAsync(["seed"], null, null, options: new IdeaOptions());
        Assert.Equal(60, unlimited.Count);
        var tool = new KeywordPlannerTool(client);
        var json = limit == 50
            ? await tool.GenerateKeywordIdeasAsync("seed")
            : await tool.GenerateKeywordIdeasAsync("seed", limit: limit);
        var result = JsonSerializer.Deserialize(json, KwpJsonContext.Default.KeywordIdeasResponse)!;
        Assert.Equal(expected, result.Count);
        Assert.Equal(60, result.TotalFound);
        using var document = JsonDocument.Parse(json);
        Assert.False(document.RootElement.GetProperty("ideas")[0].TryGetProperty("monthlySearchVolumes", out _));
        if (limit == 50)
            Assert.Equal(60, (await client.GenerateKeywordIdeasAsync(["seed"], null, null)).Count);
    }

    [Fact]
    public async Task Ideas_NegativeMinimumIsRejectedBeforeHttp()
    {
        var handler = new Handler();
        var client = Client(handler, new Pacer());
        await Assert.ThrowsAsync<ArgumentException>(() =>
            client.GenerateKeywordIdeasAsync(["seed"], null, null, options: new IdeaOptions(MinMonthlySearches: -1)));
        Assert.Empty(handler.Requests);
        var error = await new KeywordPlannerTool(client).GenerateKeywordIdeasAsync("seed", minMonthlySearches: -1);
        Assert.Contains("must not be negative", error);
    }

    [Fact]
    public async Task Targeting_CachesSuccessfulIsoAndNameLookupsDeduplicatesAliasesAndHonorsOverrides()
    {
        var handler = new Handler
        {
            Respond = (_, body) => body.Contains("language_constant", StringComparison.Ordinal)
                ? Response("""{"results":[{"languageConstant":{"id":"1001","code":"de","name":"German"}}]}""")
                : body.Contains("geo_target_constant", StringComparison.Ordinal)
                    ? Response("""{"results":[{"geoTargetConstant":{"id":"2040","countryCode":"AT","name":"Austria"}}]}""")
                    : Response("""{"results":[]}""")
        };
        var pacer = new Pacer();
        var client = Client(handler, pacer, "de", ["AT"]);
        var first = await client.GenerateKeywordIdeasAsync(["seed"], null, null);
        var second = await client.GenerateKeywordIdeasAsync(["seed"], null, " German ",
            geoTargetConstants: ["Austria", " at ", "2040", "geoTargetConstants/2040"]);
        Assert.Equal(first.Language, second.Language);
        Assert.Equal(new[] { "geoTargetConstants/2040" }, second.GeoTargetConstants);
        Assert.Equal(2, handler.Requests.Count(r => r.Address.EndsWith("/googleAds:search", StringComparison.Ordinal)));
        Assert.Contains("target_type = 'Country'", handler.Requests[1].Body);
        Assert.Contains("status = 'ENABLED'", handler.Requests[1].Body);
        Assert.Equal(2, pacer.Calls);
        var overridden = await client.GenerateKeywordIdeasAsync(["seed"], null, "all", geoTargetConstants: []);
        Assert.Null(overridden.Language);
        Assert.Null(overridden.GeoTargetConstants);
        Assert.Equal(5, handler.Requests.Count);
    }

    [Fact]
    public async Task Targeting_FailedLookupIsRetriedAndUnknownInputsAreRejected()
    {
        var attempts = 0;
        var handler = new Handler
        {
            Respond = (_, body) => body.Contains("language_constant", StringComparison.Ordinal)
                ? ++attempts == 1 ? Response("transient", HttpStatusCode.ServiceUnavailable)
                    : Response("""{"results":[{"languageConstant":{"id":"1001","code":"de","name":"German"}}]}""")
                : Response("""{"results":[]}""")
        };
        var client = Client(handler, new Pacer());
        await Assert.ThrowsAsync<InvalidOperationException>(() => client.GetHistoricalMetricsAsync(["a"], language: "de"));
        await client.GetHistoricalMetricsAsync(["a"], language: "de");
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetHistoricalMetricsAsync(["a"], language: "Klingon"));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetHistoricalMetricsAsync(["a"], geoTargetConstants: ["Atlantis"]));
        await Assert.ThrowsAsync<ArgumentException>(() => client.GetHistoricalMetricsAsync(["a"], language: " "));
        Assert.Equal(2, attempts);
    }

    [Fact]
    public async Task Targeting_ConcurrentLookupsShareCacheAndWaitingCallerCanCancel()
    {
        var entered = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        var release = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        var searches = 0;
        using var handler = new Handler
        {
            RespondAsync = async (_, body, token) =>
            {
                if (!body.Contains("language_constant", StringComparison.Ordinal))
                    return Response("""{"results":[]}""");
                Interlocked.Increment(ref searches);
                entered.TrySetResult();
                await release.Task.WaitAsync(token);
                return Response("""{"results":[{"languageConstant":{"id":"1001","code":"de","name":"German"}}]}""");
            }
        };
        using var client = Client(handler, new Pacer());
        var first = client.GetHistoricalMetricsAsync(["a"], language: "de");
        await entered.Task.WaitAsync(TimeSpan.FromSeconds(5));
        using var cancellation = new CancellationTokenSource();
        var cancelled = client.GetHistoricalMetricsAsync(["b"], cancellation.Token, language: "German");
        var queued = client.GetHistoricalMetricsAsync(["c"], language: "German");
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => cancelled);
        Assert.Equal(1, searches);
        release.TrySetResult();
        await Task.WhenAll(first, queued).WaitAsync(TimeSpan.FromSeconds(5));
        Assert.Equal(1, searches);
        Assert.Equal(3, handler.Requests.Count);
    }

    [Fact]
    public async Task Targeting_CancelledNetworkLookupIsNotCachedAndNextCallRetries()
    {
        var entered = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        var searches = 0;
        using var handler = new Handler
        {
            RespondAsync = async (_, body, token) =>
            {
                if (!body.Contains("geo_target_constant", StringComparison.Ordinal))
                    return Response("""{"results":[]}""");
                if (Interlocked.Increment(ref searches) == 1)
                {
                    entered.TrySetResult();
                    await new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously).Task.WaitAsync(token);
                }
                return Response("""{"results":[{"geoTargetConstant":{"id":"2040","countryCode":"AT","name":"Austria"}}]}""");
            }
        };
        using var client = Client(handler, new Pacer());
        using var cancellation = new CancellationTokenSource();
        var cancelled = client.GetHistoricalMetricsAsync(["a"], cancellation.Token, geoTargetConstants: ["AT"]);
        await entered.Task.WaitAsync(TimeSpan.FromSeconds(5));
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => cancelled);
        await client.GetHistoricalMetricsAsync(["a"], geoTargetConstants: ["Austria"]);
        Assert.Equal(2, searches);
    }

    [Fact]
    public async Task GeoSuggestions_UseGlobalEndpointDefaultsAndTypedResponseWithoutPlanningPace()
    {
        var handler = new Handler
        {
            Respond = (_, _) => Response("""
                {"geoTargetConstantSuggestions":[{"reach":"123456","geoTargetConstant":{
                "resourceName":"geoTargetConstants/1000001","name":"Vienna","canonicalName":"Vienna,Austria",
                "countryCode":"AT","targetType":"City"}}]}
                """)
        };
        var pacer = new Pacer();
        var tool = new KeywordPlannerTool(Client(handler, pacer));
        var result = JsonSerializer.Deserialize(await tool.SearchGeoTargetsAsync(" Vienna, Salzburg ", "at"),
            KwpJsonContext.Default.GeoTargetSearchResponse)!;
        var target = Assert.Single(result.GeoTargets);
        Assert.Equal(1, result.Count);
        Assert.Equal(new GeoTarget("geoTargetConstants/1000001", "Vienna", "Vienna,Austria", "AT", "City", 123456), target);
        var request = Assert.Single(handler.Requests);
        Assert.Equal("https://googleads.googleapis.com/v25/geoTargetConstants:suggest", request.Address);
        using var json = JsonDocument.Parse(request.Body);
        Assert.Equal("en", json.RootElement.GetProperty("locale").GetString());
        Assert.Equal("AT", json.RootElement.GetProperty("countryCode").GetString());
        Assert.Equal(2, json.RootElement.GetProperty("locationNames").GetProperty("names").GetArrayLength());
        Assert.Equal(0, pacer.Calls);
        Assert.Contains("nonblank", await tool.SearchGeoTargetsAsync("Vienna, "));
        Assert.Single(handler.Requests);
    }
}
