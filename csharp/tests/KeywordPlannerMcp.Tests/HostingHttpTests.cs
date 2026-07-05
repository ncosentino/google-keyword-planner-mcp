using System.Net;
using System.Text;
using Microsoft.AspNetCore.Builder;
using ModelContextProtocol.Client;
using Xunit;

namespace KeywordPlannerMcp.Tests;

/// <summary>
/// Tests for the HTTP transport host (Hosting.BuildHttpHost), exercising a real
/// bound Kestrel instance and a real MCP client connecting over HTTP -- the same
/// wiring Program.cs uses for --transport http, minus argument parsing.
/// </summary>
public sealed class HostingHttpTests
{
    private const string DevToken = "dev-token";
    private const string ClientId = "client-id";
    private const string ClientSecret = "client-secret";
    private const string RefreshToken = "refresh-token";
    private const string CustomerId = "3778350596";

    /// <summary>
    /// Fake handler that satisfies the OAuth2 token exchange with a canned success
    /// response, then returns a configurable body for the Google Ads API call itself.
    /// Used to exercise get_historical_metrics/get_keyword_forecast end-to-end through
    /// a real MCP session without making any real network call.
    /// </summary>
    private sealed class FakeGoogleAdsHandler(string apiResponseBody) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            if (request.RequestUri?.Host == "oauth2.googleapis.com")
            {
                const string tokenJson = """{"access_token":"fake-token","expires_in":3600}""";
                return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
                {
                    Content = new StringContent(tokenJson, Encoding.UTF8, "application/json"),
                });
            }

            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(apiResponseBody, Encoding.UTF8, "application/json"),
            });
        }
    }

    /// <summary>
    /// Hosting.BuildHttpHost binds "0.0.0.0" (all interfaces), which is correct for
    /// production but isn't itself a connectable target address -- app.Urls reports
    /// back the bind address verbatim. Tests connecting from the same machine need
    /// to target loopback explicitly, on whatever port Kestrel actually chose.
    /// </summary>
    private static Uri ConnectableUri(WebApplication app)
    {
        var bound = new Uri(app.Urls.First());
        return new UriBuilder(bound) { Host = "127.0.0.1" }.Uri;
    }

    [Fact]
    public async Task BuildHttpHost_ServesRealSession_ListsAllTools()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            await using var client = await McpClient.CreateAsync(new HttpClientTransport(new HttpClientTransportOptions
            {
                Endpoint = ConnectableUri(app),
            }));

            var tools = await client.ListToolsAsync();

            Assert.Equal(3, tools.Count);
            Assert.Contains(tools, t => t.Name == "generate_keyword_ideas");
            Assert.Contains(tools, t => t.Name == "get_historical_metrics");
            Assert.Contains(tools, t => t.Name == "get_keyword_forecast");
        }
        finally
        {
            await app.StopAsync();
        }
    }

    [Fact]
    public async Task BuildHttpHost_CallToolOverHttp_ReturnsValidationErrorWithoutCallingGoogleAds()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            var baseUri = ConnectableUri(app);
            await using var client = await McpClient.CreateAsync(new HttpClientTransport(new HttpClientTransportOptions
            {
                Endpoint = baseUri,
            }));

            // No seedKeywords or url: this must hit the validation short-circuit in
            // KeywordPlannerTool rather than actually calling the Google Ads API (which
            // would fail anyway, since DevToken/ClientId/etc. above aren't real).
            var result = await client.CallToolAsync("generate_keyword_ideas", new Dictionary<string, object?>());

            var text = Assert.IsType<ModelContextProtocol.Protocol.TextContentBlock>(Assert.Single(result.Content));
            Assert.Contains("At least one of seedKeywords or url must be provided", text.Text, StringComparison.Ordinal);
        }
        finally
        {
            await app.StopAsync();
        }
    }

    [Fact]
    public async Task BuildHttpHost_NoAllowedHostsConfigured_DefaultsToLoopback()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);

        Assert.Equal(Hosting.DefaultAllowedHosts, app.Configuration["AllowedHosts"]);
    }

    [Fact]
    public async Task BuildHttpHost_AllowedHostsPassedOnCommandLine_OverridesDefault()
    {
        await using var app = Hosting.BuildHttpHost(
            ["--AllowedHosts", "example.com"],
            DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);

        Assert.Equal("example.com", app.Configuration["AllowedHosts"]);
    }

    /// <summary>
    /// Confirms get_historical_metrics -- previously exercised only via direct C#
    /// method calls in KeywordPlannerToolTests, never through a real MCP session --
    /// works end-to-end: argument binding (comma-separated "keywords" string),
    /// [McpServerTool] reflection-based dispatch, and the DI-registered
    /// KeywordPlannerClient all have to agree for this to pass.
    /// </summary>
    [Fact]
    public async Task BuildHttpHost_CallHistoricalMetricsTool_ViaRealSession_ReturnsSuccessResult()
    {
        var handler = new FakeGoogleAdsHandler("""{"metrics":[]}""");
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0, handler);
        await app.StartAsync();
        try
        {
            await using var client = await McpClient.CreateAsync(new HttpClientTransport(new HttpClientTransportOptions
            {
                Endpoint = ConnectableUri(app),
            }));

            var result = await client.CallToolAsync(
                "get_historical_metrics", new Dictionary<string, object?> { ["keywords"] = "dependency injection" });

            // IsError is nullable: null (unset) on success, true on error.
            Assert.NotEqual(true, result.IsError);
        }
        finally
        {
            await app.StopAsync();
        }
    }

    /// <summary>
    /// get_keyword_forecast equivalent of
    /// BuildHttpHost_CallHistoricalMetricsTool_ViaRealSession_ReturnsSuccessResult.
    /// </summary>
    [Fact]
    public async Task BuildHttpHost_CallKeywordForecastTool_ViaRealSession_ReturnsSuccessResult()
    {
        var handler = new FakeGoogleAdsHandler("""{"adGroupForecastMetrics":[]}""");
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0, handler);
        await app.StartAsync();
        try
        {
            await using var client = await McpClient.CreateAsync(new HttpClientTransport(new HttpClientTransportOptions
            {
                Endpoint = ConnectableUri(app),
            }));

            var result = await client.CallToolAsync(
                "get_keyword_forecast", new Dictionary<string, object?> { ["keywords"] = "dependency injection" });

            // IsError is nullable: null (unset) on success, true on error.
            Assert.NotEqual(true, result.IsError);
        }
        finally
        {
            await app.StopAsync();
        }
    }
}
