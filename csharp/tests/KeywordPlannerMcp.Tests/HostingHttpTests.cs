using System.Net;
using System.Text;
using System.Text.Json;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;
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
    /// Returns a loopback URL for the requested endpoint on Kestrel's selected port.
    /// </summary>
    private static Uri ConnectableUri(WebApplication app, string path = Hosting.McpPath)
    {
        var bound = new Uri(app.Urls.First());
        return new UriBuilder(bound)
        {
            Host = ServerOptions.DefaultListenAddress,
            Path = path,
        }.Uri;
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
    public async Task BuildHttpHost_DefaultsToLoopbackBinding()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            var bound = new Uri(app.Urls.First());
            Assert.Equal(ServerOptions.DefaultListenAddress, bound.Host);
        }
        finally
        {
            await app.StopAsync();
        }
    }

    [Fact]
    public async Task BuildHttpHost_ServesHealth()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            using var httpClient = new HttpClient();
            using var response = await httpClient.GetAsync(ConnectableUri(app, Hosting.HealthPath));
            response.EnsureSuccessStatusCode();

            using var document = JsonDocument.Parse(await response.Content.ReadAsStringAsync());
            Assert.Equal("ok", document.RootElement.GetProperty("status").GetString());
            Assert.Equal(
                "google-keyword-planner-mcp",
                document.RootElement.GetProperty("service").GetString());
            Assert.Equal("http", document.RootElement.GetProperty("transport").GetString());
        }
        finally
        {
            await app.StopAsync();
        }
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

    [Fact]
    public async Task BuildHttpHost_RejectsCrossSiteOrigin()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            using var httpClient = new HttpClient();
            using var request = new HttpRequestMessage(
                HttpMethod.Post,
                ConnectableUri(app, Hosting.McpPath));
            request.Headers.Add("Origin", "https://evil.example");
            request.Content = new StringContent("{}", Encoding.UTF8, "application/json");

            using var response = await httpClient.SendAsync(request);

            Assert.Equal(HttpStatusCode.Forbidden, response.StatusCode);
        }
        finally
        {
            await app.StopAsync();
        }
    }

    [Fact]
    public async Task BuildHttpHost_RejectsCrossSiteOriginWithTrailingSlash()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            using var httpClient = new HttpClient();
            using var request = new HttpRequestMessage(
                HttpMethod.Post,
                ConnectableUri(app, Hosting.McpPath + "/"));
            request.Headers.Add("Origin", "https://evil.example");
            request.Content = new StringContent("{}", Encoding.UTF8, "application/json");

            using var response = await httpClient.SendAsync(request);

            Assert.Equal(HttpStatusCode.Forbidden, response.StatusCode);
        }
        finally
        {
            await app.StopAsync();
        }
    }

    [Fact]
    public void IsCrossOriginRequestAllowed_AcceptsSameOrigin()
    {
        var context = new DefaultHttpContext();
        context.Request.Method = HttpMethods.Post;
        context.Request.Scheme = "http";
        context.Request.Host = new HostString("127.0.0.1", 8080);
        context.Request.Headers.Origin = "http://127.0.0.1:8080";

        Assert.True(Hosting.IsCrossOriginRequestAllowed(context.Request));
    }

    [Fact]
    public async Task BuildHttpHost_RejectsCrossSiteFetchMetadataWithoutOrigin()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            using var httpClient = new HttpClient();
            using var request = new HttpRequestMessage(
                HttpMethod.Post,
                ConnectableUri(app, Hosting.McpPath));
            request.Headers.Add("Sec-Fetch-Site", "cross-site");
            request.Content = new StringContent("{}", Encoding.UTF8, "application/json");

            using var response = await httpClient.SendAsync(request);

            Assert.Equal(HttpStatusCode.Forbidden, response.StatusCode);
        }
        finally
        {
            await app.StopAsync();
        }
    }

    [Fact]
    public async Task BuildHttpHost_AllowsSafeCrossSiteMethod()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            using var httpClient = new HttpClient();
            using var request = new HttpRequestMessage(
                HttpMethod.Get,
                ConnectableUri(app, Hosting.McpPath));
            request.Headers.Add("Sec-Fetch-Site", "cross-site");
            request.Headers.Add("Origin", "https://evil.example");

            using var response = await httpClient.SendAsync(request);

            Assert.NotEqual(HttpStatusCode.Forbidden, response.StatusCode);
        }
        finally
        {
            await app.StopAsync();
        }
    }

    [Fact]
    public async Task BuildHttpHost_RejectsDisallowedHost()
    {
        await using var app = Hosting.BuildHttpHost(
            [], DevToken, ClientId, ClientSecret, RefreshToken, CustomerId, loginCustomerId: null, port: 0);
        await app.StartAsync();
        try
        {
            using var httpClient = new HttpClient();
            using var request = new HttpRequestMessage(
                HttpMethod.Get,
                ConnectableUri(app, Hosting.HealthPath));
            request.Headers.Host = "evil.example";

            using var response = await httpClient.SendAsync(request);

            Assert.Equal(HttpStatusCode.BadRequest, response.StatusCode);
        }
        finally
        {
            await app.StopAsync();
        }
    }

    [Fact]
    public async Task BuildHttpHost_ShutdownRequiresBearerToken()
    {
        await using var app = Hosting.BuildHttpHost(
            [],
            DevToken,
            ClientId,
            ClientSecret,
            RefreshToken,
            CustomerId,
            loginCustomerId: null,
            port: 0,
            shutdownToken: "secret-token");
        await app.StartAsync();

        using var httpClient = new HttpClient();
        using var rejectedRequest = new HttpRequestMessage(
            HttpMethod.Post,
            ConnectableUri(app, Hosting.ShutdownPath));
        rejectedRequest.Headers.Authorization =
            new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", "wrong-token");
        using var rejectedResponse = await httpClient.SendAsync(rejectedRequest);
        Assert.Equal(HttpStatusCode.Unauthorized, rejectedResponse.StatusCode);

        var stopping = new TaskCompletionSource<bool>(
            TaskCreationOptions.RunContinuationsAsynchronously);
        app.Lifetime.ApplicationStopping.Register(() => stopping.SetResult(true));
        using var acceptedRequest = new HttpRequestMessage(
            HttpMethod.Post,
            ConnectableUri(app, Hosting.ShutdownPath));
        acceptedRequest.Headers.Authorization =
            new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", "secret-token");
        using var acceptedResponse = await httpClient.SendAsync(acceptedRequest);

        Assert.Equal(HttpStatusCode.Accepted, acceptedResponse.StatusCode);
        await stopping.Task.WaitAsync(TimeSpan.FromSeconds(5));
    }
}
