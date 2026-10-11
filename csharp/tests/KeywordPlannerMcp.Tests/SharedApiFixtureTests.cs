using System.Net;
using System.Text.Json;
using System.Text.Json.Nodes;
using KeywordPlannerMcp.KeywordPlanner;
using Xunit;

namespace KeywordPlannerMcp.Tests;

public sealed class SharedApiFixtureTests
{
    private sealed class FixtureHandler(string fixture) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken) =>
            Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(request.RequestUri!.Host == "oauth2.googleapis.com"
                    ? """{"access_token":"fake","expires_in":3600}"""
                    : File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "fixtures", fixture)))
            });
    }

    private sealed class ImmediatePacer : IPlanningPacer
    {
        public Task WaitAsync(CancellationToken cancellationToken)
        {
            cancellationToken.ThrowIfCancellationRequested();
            return Task.CompletedTask;
        }
    }

    private static KeywordPlannerClient CreateClient(string fixture)
    {
        var http = new HttpClient(new FixtureHandler(fixture));
        return new KeywordPlannerClient("token", "123", null,
            new OAuth2TokenProvider("id", "secret", "refresh", http), http,
            planningPacer: new ImmediatePacer());
    }

    [Fact]
    public async Task SharedHistoricalFixture()
    {
        using var client = CreateClient("historical.json");
        var result = await client.GetHistoricalMetricsAsync(["dependency injection"]);
        var keyword = Assert.Single(result.Keywords);
        Assert.Equal(5000, keyword.AvgMonthlySearches);
        Assert.Equal(23, keyword.CompetitionIndex);
        Assert.Equal(new MonthlyVolume(2026, 9, 4500), Assert.Single(keyword.MonthlySearchVolumes));
    }

    [Fact]
    public async Task SharedIdeaFixture()
    {
        using var client = CreateClient("ideas.json");
        var result = await client.GenerateKeywordIdeasAsync(["go"], null, null,
            options: new IdeaOptions(IncludeMonthlyVolumes: true));
        var idea = Assert.Single(result.Ideas);
        Assert.Equal(1, result.TotalFound);
        Assert.Equal(5000, idea.AvgMonthlySearches);
        Assert.Equal(new MonthlyVolume(2026, 9, 4500), Assert.Single(idea.MonthlySearchVolumes!));
        using var json = JsonDocument.Parse(JsonSerializer.Serialize(result, KwpJsonContext.Default.KeywordIdeasResponse));
        Assert.Equal(0, json.RootElement.GetProperty("ideas")[0].GetProperty("competitionIndex").GetInt32());
    }

    private sealed class RequestHandler : HttpMessageHandler
    {
        public JsonNode? Request { get; private set; }
        protected override async Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            var oauth = request.RequestUri!.Host == "oauth2.googleapis.com";
            if (!oauth)
                Request = JsonNode.Parse(await request.Content!.ReadAsStringAsync(cancellationToken));
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(oauth
                    ? """{"access_token":"fake","expires_in":3600}"""
                    : File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "fixtures", "forecast.json")))
            };
        }
    }

    private sealed class FixedClock : TimeProvider
    {
        public override DateTimeOffset GetUtcNow() => new(2026, 10, 11, 0, 0, 0, TimeSpan.Zero);
    }

    [Fact]
    public async Task SharedForecastRequestFixture()
    {
        using var handler = new RequestHandler();
        using var http = new HttpClient(handler);
        using var provider = new OAuth2TokenProvider("id", "secret", "refresh", http);
        using var client = new KeywordPlannerClient("token", "123", null, provider, http,
            timeProvider: new FixedClock(), planningPacer: new ImmediatePacer());
        await client.GetKeywordForecastAsync(["go"], 2000000, 30,
            language: "languageConstants/1001", geoTargetConstants: ["geoTargetConstants/2040"],
            matchType: "PHRASE", currencyCode: "EUR");
        var period = handler.Request!["forecastPeriod"]!;
        Assert.Equal("2026-10-13", period["startDate"]!.GetValue<string>());
        Assert.Equal("2026-11-11", period["endDate"]!.GetValue<string>());
        period["startDate"] = "$start";
        period["endDate"] = "$end";
        var expected = JsonNode.Parse(File.ReadAllText(Path.Combine(
            AppContext.BaseDirectory, "fixtures", "forecast-request.json")));
        Assert.True(JsonNode.DeepEquals(expected, handler.Request));
    }

    [Theory]
    [InlineData("forecast.json", true, 12.5, 9000000000L)]
    [InlineData("forecast-zero.json", true, 0, 0L)]
    [InlineData("forecast-unavailable.json", false, 0, 0L)]
    public async Task SharedForecastFixtures(string fixture, bool present, double clicks, long cost)
    {
        using var client = CreateClient(fixture);
        var result = await client.GetKeywordForecastAsync(["go"], 1000000, 30);
        Assert.Equal(present, result.Total.Clicks.HasValue);
        Assert.Equal(present, result.Total.CostMicros.HasValue);
        if (present)
        {
            Assert.Equal(clicks, result.Total.Clicks);
            Assert.Equal(cost, result.Total.CostMicros);
        }
    }
}
