using System.Net;
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
