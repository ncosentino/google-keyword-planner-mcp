using System.Net;
using System.Reflection;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using KeywordPlannerMcp.KeywordPlanner;
using KeywordPlannerMcp.Tools;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Primitives;
using ModelContextProtocol.Server;

namespace KeywordPlannerMcp;

/// <summary>Builds the MCP server's hosts for both the stdio and HTTP transports.</summary>
internal static class Hosting
{
    /// <summary>Default Host header allow-list when none is configured via AllowedHosts.</summary>
    internal const string DefaultAllowedHosts = "localhost;127.0.0.1;[::1]";

    /// <summary>Health-check endpoint for service supervisors.</summary>
    internal const string HealthPath = "/health";

    /// <summary>Streamable HTTP MCP endpoint.</summary>
    internal const string McpPath = "/mcp";

    /// <summary>Authenticated local service shutdown endpoint.</summary>
    internal const string ShutdownPath = "/shutdown";

    private const long MaxMcpRequestBytes = 1 << 20;

    /// <summary>Builds an HTTP host without starting it.</summary>
    internal static WebApplication BuildHttpHost(
        string[] args,
        string developerToken,
        string clientId,
        string clientSecret,
        string refreshToken,
        string customerId,
        string? loginCustomerId,
        int port,
        HttpMessageHandler? httpMessageHandler = null,
        string listenAddress = ServerOptions.DefaultListenAddress,
        string? shutdownToken = null)
    {
        var builder = WebApplication.CreateBuilder(args);
        if (string.IsNullOrWhiteSpace(builder.Configuration["AllowedHosts"]))
        {
            builder.Configuration["AllowedHosts"] = DefaultAllowedHosts;
        }

        builder.WebHost.UseUrls($"http://{FormatListenAddress(listenAddress)}:{port}");
        builder.WebHost.ConfigureKestrel(options =>
        {
            options.Limits.MaxRequestBodySize = MaxMcpRequestBytes;
            options.Limits.RequestHeadersTimeout = TimeSpan.FromSeconds(5);
            options.Limits.KeepAliveTimeout = TimeSpan.FromMinutes(2);
        });

        ConfigureCommonServices(
            builder,
            developerToken,
            clientId,
            clientSecret,
            refreshToken,
            customerId,
            loginCustomerId,
            httpMessageHandler);
        builder.Services.AddMcpServer()
            .WithHttpTransport(options => options.Stateless = true)
            .WithTools<KeywordPlannerTool>();

        var app = builder.Build();
        app.Use(async (context, next) =>
        {
            if (context.Request.Path.StartsWithSegments(McpPath) &&
                !IsCrossOriginRequestAllowed(context.Request))
            {
                context.Response.StatusCode = StatusCodes.Status403Forbidden;
                return;
            }
            await next(context).ConfigureAwait(false);
        });
        app.MapGet(HealthPath, () =>
        {
            var response = new ServiceHealth(
                "ok",
                "google-keyword-planner-mcp",
                GetServiceVersion(),
                "http");
            return Results.Text(
                JsonSerializer.Serialize(response, HostingJsonContext.Default.ServiceHealth),
                "application/json");
        });
        app.MapMcp(McpPath);
        if (!string.IsNullOrEmpty(shutdownToken))
        {
            app.MapPost(ShutdownPath, async context =>
            {
                if (context.Connection.RemoteIpAddress is null ||
                    !IPAddress.IsLoopback(context.Connection.RemoteIpAddress))
                {
                    context.Response.StatusCode = StatusCodes.Status403Forbidden;
                    return;
                }
                if (!HasBearerToken(context.Request, shutdownToken))
                {
                    context.Response.StatusCode = StatusCodes.Status401Unauthorized;
                    return;
                }

                context.Response.StatusCode = StatusCodes.Status202Accepted;
                context.Response.ContentType = "application/json";
                context.Response.Headers.CacheControl = "no-store";
                await context.Response.WriteAsync("""{"stopping":true}""")
                    .ConfigureAwait(false);
                app.Lifetime.StopApplication();
            });
        }
        return app;
    }

    /// <summary>Registers services shared by both transports.</summary>
    internal static void ConfigureCommonServices(
        IHostApplicationBuilder builder,
        string developerToken,
        string clientId,
        string clientSecret,
        string refreshToken,
        string customerId,
        string? loginCustomerId,
        HttpMessageHandler? httpMessageHandler = null)
    {
        builder.Logging.AddConsole(options =>
            options.LogToStandardErrorThreshold = LogLevel.Trace);
        builder.Logging.SetMinimumLevel(LogLevel.Warning);

        var oauthClient = builder.Services.AddHttpClient<OAuth2TokenProvider>(http =>
        {
            http.Timeout = TimeSpan.FromSeconds(30);
        });
        var apiClient = builder.Services.AddHttpClient<KeywordPlannerClient>(http =>
        {
            http.Timeout = TimeSpan.FromSeconds(30);
        });
        if (httpMessageHandler is not null)
        {
            oauthClient.ConfigurePrimaryHttpMessageHandler(() => httpMessageHandler);
            apiClient.ConfigurePrimaryHttpMessageHandler(() => httpMessageHandler);
        }

        builder.Services.AddTransient<OAuth2TokenProvider>(services =>
        {
            var factory = services.GetRequiredService<IHttpClientFactory>();
            return new OAuth2TokenProvider(
                clientId,
                clientSecret,
                refreshToken,
                factory.CreateClient(nameof(OAuth2TokenProvider)));
        });
        builder.Services.AddTransient<KeywordPlannerClient>(services =>
        {
            var factory = services.GetRequiredService<IHttpClientFactory>();
            var tokenProvider = services.GetRequiredService<OAuth2TokenProvider>();
            return new KeywordPlannerClient(
                developerToken,
                customerId,
                loginCustomerId,
                tokenProvider,
                factory.CreateClient(nameof(KeywordPlannerClient)));
        });
    }

    internal static bool IsCrossOriginRequestAllowed(HttpRequest request)
    {
        if (HttpMethods.IsGet(request.Method) ||
            HttpMethods.IsHead(request.Method) ||
            HttpMethods.IsOptions(request.Method))
        {
            return true;
        }

        var fetchSite = request.Headers["Sec-Fetch-Site"].ToString();
        if (fetchSite is "same-origin" or "none")
        {
            return true;
        }
        if (fetchSite.Length != 0)
        {
            return false;
        }

        StringValues origins = request.Headers.Origin;
        if (StringValues.IsNullOrEmpty(origins))
        {
            return true;
        }
        if (origins.Count != 1 ||
            !Uri.TryCreate(origins[0], UriKind.Absolute, out var origin) ||
            origin.UserInfo.Length != 0)
        {
            return false;
        }

        return string.Equals(
            origin.Authority,
            request.Host.Value,
            StringComparison.OrdinalIgnoreCase);
    }

    private static bool HasBearerToken(HttpRequest request, string expectedToken)
    {
        const string prefix = "Bearer ";
        var authorization = request.Headers.Authorization.ToString();
        if (!authorization.StartsWith(prefix, StringComparison.Ordinal))
        {
            return false;
        }

        return CryptographicOperations.FixedTimeEquals(
            Encoding.UTF8.GetBytes(authorization[prefix.Length..]),
            Encoding.UTF8.GetBytes(expectedToken));
    }

    private static string FormatListenAddress(string listenAddress) =>
        IPAddress.TryParse(listenAddress, out var address) &&
        address.AddressFamily == System.Net.Sockets.AddressFamily.InterNetworkV6
            ? $"[{listenAddress}]"
            : listenAddress;

    private static string GetServiceVersion()
    {
        var assembly = typeof(Hosting).Assembly;
        return assembly.GetCustomAttribute<AssemblyInformationalVersionAttribute>()?
            .InformationalVersion
            ?? assembly.GetName().Version?.ToString()
            ?? "dev";
    }
}

internal sealed record ServiceHealth(
    string Status,
    string Service,
    string Version,
    string Transport);

/// <summary>System.Text.Json source generation context for service metadata.</summary>
[JsonSerializable(typeof(ServiceHealth))]
[JsonSourceGenerationOptions(PropertyNamingPolicy = JsonKnownNamingPolicy.CamelCase)]
internal partial class HostingJsonContext : JsonSerializerContext;
