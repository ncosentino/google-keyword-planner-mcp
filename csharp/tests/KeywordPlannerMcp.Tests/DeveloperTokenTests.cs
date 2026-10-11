using System.Net;
using KeywordPlannerMcp.KeywordPlanner;
using Xunit;

namespace KeywordPlannerMcp.Tests;

public sealed class DeveloperTokenTests
{
    private sealed class Handler : HttpMessageHandler
    {
        public bool ApiCalled { get; private set; }
        protected override Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            var tokenRequest = request.RequestUri!.Host == "oauth2.googleapis.com";
            if (!tokenRequest)
            {
                ApiCalled = true;
                Assert.False(request.Headers.Contains("developer-token"));
            }
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(tokenRequest
                    ? """{"access_token":"fake","expires_in":3600}"""
                    : """{"results":[]}""")
            });
        }
    }

    [Fact]
    public async Task ClientWorksWithoutLegacyDeveloperToken()
    {
        var handler = new Handler();
        using var http = new HttpClient(handler);
        using var provider = new OAuth2TokenProvider("id", "secret", "refresh", http);
        using var client = new KeywordPlannerClient("", "123", null, provider, http);
        var result = await client.GenerateKeywordIdeasAsync(["go"], null, null);
        Assert.Empty(result.Ideas);
        Assert.True(handler.ApiCalled);
    }
}
