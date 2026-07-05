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
}
