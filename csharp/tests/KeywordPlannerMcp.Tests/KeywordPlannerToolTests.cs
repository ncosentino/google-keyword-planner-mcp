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

    private static KeywordPlannerTool CreateToolWithUnreachableClient()
    {
        var handler = new UnreachableHttpHandler();
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
}
