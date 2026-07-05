using System.Net;
using System.Text;
using System.Text.Json;
using KeywordPlannerMcp.KeywordPlanner;
using KeywordPlannerMcp.Tools;
using Xunit;

namespace KeywordPlannerMcp.Tests;

/// <summary>
/// Tests for KeywordPlannerTool's own validation logic, distinct from
/// KeywordPlannerClientTests (which covers the HTTP-calling client).
/// </summary>
public sealed class KeywordPlannerToolTests
{
    /// <summary>Fake handler that fails the test if any HTTP request reaches it.</summary>
    private sealed class UnreachableHttpHandler : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            throw new InvalidOperationException(
                $"Unexpected HTTP request to {request.RequestUri}; the Google Ads API must not be " +
                "called when seedKeywords and url are both missing.");
        }
    }

    /// <summary>
    /// Fake handler that satisfies the OAuth2 token exchange with a canned success
    /// response, then returns a configurable status/body for the Google Ads API call
    /// itself -- so tool-level success and error paths can both be exercised without
    /// any real network call.
    /// </summary>
    private sealed class FakeGoogleAdsHandler(HttpStatusCode apiStatusCode, string apiResponseBody) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            if (request.RequestUri?.Host == "oauth2.googleapis.com")
            {
                var tokenJson = JsonSerializer.Serialize(new
                {
                    access_token = "fake-token",
                    expires_in = 3600,
                    token_type = "Bearer"
                });
                return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
                {
                    Content = new StringContent(tokenJson, Encoding.UTF8, "application/json"),
                });
            }

            return Task.FromResult(new HttpResponseMessage(apiStatusCode)
            {
                Content = new StringContent(apiResponseBody, Encoding.UTF8, "application/json"),
            });
        }
    }

    private static KeywordPlannerTool CreateToolWithUnreachableClient()
    {
        var handler = new UnreachableHttpHandler();
        var tokenProvider = new OAuth2TokenProvider("client-id", "client-secret", "refresh-token", new HttpClient(handler));
        var client = new KeywordPlannerClient("dev-token", "3778350596", null, tokenProvider, new HttpClient(handler));
        return new KeywordPlannerTool(client);
    }

    private static KeywordPlannerTool CreateToolWithFakeApi(HttpStatusCode apiStatusCode, string apiResponseBody)
    {
        var handler = new FakeGoogleAdsHandler(apiStatusCode, apiResponseBody);
        var tokenProvider = new OAuth2TokenProvider("client-id", "client-secret", "refresh-token", new HttpClient(handler));
        var client = new KeywordPlannerClient("dev-token", "3778350596", null, tokenProvider, new HttpClient(handler));
        return new KeywordPlannerTool(client);
    }

    [Fact]
    public async Task GenerateKeywordIdeasAsync_NoSeedKeywordsOrUrl_ReturnsValidationError()
    {
        var tool = CreateToolWithUnreachableClient();

        var resultJson = await tool.GenerateKeywordIdeasAsync();

        var result = JsonSerializer.Deserialize(resultJson, KwpJsonContext.Default.ErrorResult);
        Assert.NotNull(result);
        Assert.Contains("At least one of seedKeywords or url must be provided", result.Error);
    }

    [Fact]
    public async Task GenerateKeywordIdeasAsync_WhitespaceSeedKeywordsAndUrl_ReturnsValidationError()
    {
        var tool = CreateToolWithUnreachableClient();

        var resultJson = await tool.GenerateKeywordIdeasAsync(seedKeywords: "   ", url: "   ");

        var result = JsonSerializer.Deserialize(resultJson, KwpJsonContext.Default.ErrorResult);
        Assert.NotNull(result);
        Assert.Contains("At least one of seedKeywords or url must be provided", result.Error);
    }

    [Fact]
    public async Task GenerateKeywordIdeasAsync_ValidSeedKeywords_ReturnsSerializedIdeasResponse()
    {
        var tool = CreateToolWithFakeApi(HttpStatusCode.OK, """{"results":[]}""");

        var resultJson = await tool.GenerateKeywordIdeasAsync(seedKeywords: "dependency injection");

        var result = JsonSerializer.Deserialize(resultJson, KwpJsonContext.Default.KeywordIdeasResponse);
        Assert.NotNull(result);
        Assert.Equal(0, result.Count);
        Assert.Contains("dependency injection", result.SeedKeywords!);
    }

    [Fact]
    public async Task GenerateKeywordIdeasAsync_ApiError_ReturnsErrorResult()
    {
        var tool = CreateToolWithFakeApi(HttpStatusCode.InternalServerError, "boom");

        var resultJson = await tool.GenerateKeywordIdeasAsync(seedKeywords: "dependency injection");

        var result = JsonSerializer.Deserialize(resultJson, KwpJsonContext.Default.ErrorResult);
        Assert.NotNull(result);
        Assert.Contains("Error generating keyword ideas", result.Error);
    }

    [Fact]
    public async Task GetHistoricalMetricsAsync_ValidKeywords_ReturnsSerializedMetricsResponse()
    {
        var tool = CreateToolWithFakeApi(HttpStatusCode.OK, """
            {
                "metrics": [{
                    "text": "dependency injection",
                    "keywordMetrics": {
                        "avgMonthlySearches": "1000",
                        "competition": "MEDIUM",
                        "competitionIndex": 50,
                        "lowTopOfPageBidMicros": "100000",
                        "highTopOfPageBidMicros": "500000",
                        "monthlySearchVolumes": []
                    }
                }]
            }
            """);

        var resultJson = await tool.GetHistoricalMetricsAsync(keywords: "dependency injection");

        var result = JsonSerializer.Deserialize(resultJson, KwpJsonContext.Default.HistoricalMetricsResponse);
        Assert.NotNull(result);
        Assert.Equal(1, result.Count);
        Assert.Equal("dependency injection", result.Keywords[0].Text);
    }

    [Fact]
    public async Task GetHistoricalMetricsAsync_ApiError_ReturnsErrorResult()
    {
        var tool = CreateToolWithFakeApi(HttpStatusCode.InternalServerError, "boom");

        var resultJson = await tool.GetHistoricalMetricsAsync(keywords: "dependency injection");

        var result = JsonSerializer.Deserialize(resultJson, KwpJsonContext.Default.ErrorResult);
        Assert.NotNull(result);
        Assert.Contains("Error getting historical metrics", result.Error);
    }

    [Fact]
    public async Task GetKeywordForecastAsync_ValidKeywords_ReturnsSerializedForecastResponse()
    {
        var tool = CreateToolWithFakeApi(HttpStatusCode.OK, """
            {
                "adGroupForecastMetrics": [{
                    "keywordForecastMetrics": [{
                        "keyword": {"text": "dependency injection", "matchType": "BROAD"},
                        "metrics": {"impressions": 1000, "clicks": 50, "costMicros": 500000, "ctr": 0.05}
                    }]
                }]
            }
            """);

        var resultJson = await tool.GetKeywordForecastAsync(keywords: "dependency injection");

        var result = JsonSerializer.Deserialize(resultJson, KwpJsonContext.Default.ForecastResponse);
        Assert.NotNull(result);
        Assert.Single(result.Keywords);
        Assert.Equal("dependency injection", result.Keywords[0].Text);
        Assert.Equal(30, result.ForecastDays);
        Assert.Equal(1_000_000, result.MaxCpcMicros);
    }

    [Fact]
    public async Task GetKeywordForecastAsync_ApiError_ReturnsErrorResult()
    {
        var tool = CreateToolWithFakeApi(HttpStatusCode.InternalServerError, "boom");

        var resultJson = await tool.GetKeywordForecastAsync(keywords: "dependency injection");

        var result = JsonSerializer.Deserialize(resultJson, KwpJsonContext.Default.ErrorResult);
        Assert.NotNull(result);
        Assert.Contains("Error getting keyword forecast", result.Error);
    }
}
