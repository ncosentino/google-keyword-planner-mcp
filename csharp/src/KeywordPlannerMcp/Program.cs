using KeywordPlannerMcp.Config;
using KeywordPlannerMcp.KeywordPlanner;
using KeywordPlannerMcp.Tools;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using ModelContextProtocol.Server;

string? developerToken = CredentialResolver.ResolveCredential(
    args.SkipWhile(a => a != "--developer-token").Skip(1).FirstOrDefault(),
    CredentialResolver.EnvDeveloperToken);

string? clientId = CredentialResolver.ResolveCredential(
    args.SkipWhile(a => a != "--client-id").Skip(1).FirstOrDefault(),
    CredentialResolver.EnvClientId);

string? clientSecret = CredentialResolver.ResolveCredential(
    args.SkipWhile(a => a != "--client-secret").Skip(1).FirstOrDefault(),
    CredentialResolver.EnvClientSecret);

string? refreshToken = CredentialResolver.ResolveCredential(
    args.SkipWhile(a => a != "--refresh-token").Skip(1).FirstOrDefault(),
    CredentialResolver.EnvRefreshToken);

string? customerId = CredentialResolver.NormalizeCustomerId(
    CredentialResolver.ResolveCredential(
        args.SkipWhile(a => a != "--customer-id").Skip(1).FirstOrDefault(),
        CredentialResolver.EnvCustomerId));

if (string.IsNullOrWhiteSpace(developerToken) ||
    string.IsNullOrWhiteSpace(clientId) ||
    string.IsNullOrWhiteSpace(clientSecret) ||
    string.IsNullOrWhiteSpace(refreshToken) ||
    string.IsNullOrWhiteSpace(customerId))
{
    await Console.Error.WriteLineAsync(
        "ERROR: Incomplete Google Ads credentials. Required: " +
        "GOOGLE_ADS_DEVELOPER_TOKEN, GOOGLE_ADS_CLIENT_ID, GOOGLE_ADS_CLIENT_SECRET, " +
        "GOOGLE_ADS_REFRESH_TOKEN, GOOGLE_ADS_CUSTOMER_ID. " +
        "Use --flag <value>, env vars, or a .env file.")
        .ConfigureAwait(false);
    return 1;
}

var builder = Host.CreateApplicationBuilder(args);

// All logs must go to stderr to avoid corrupting the MCP STDIO stream.
builder.Logging.AddConsole(o => o.LogToStandardErrorThreshold = LogLevel.Trace);
builder.Logging.SetMinimumLevel(LogLevel.Warning);

builder.Services.AddHttpClient<OAuth2TokenProvider>(http =>
{
    http.Timeout = TimeSpan.FromSeconds(30);
});

builder.Services.AddTransient<OAuth2TokenProvider>(sp =>
{
    var factory = sp.GetRequiredService<IHttpClientFactory>();
    return new OAuth2TokenProvider(
        clientId, clientSecret, refreshToken,
        factory.CreateClient(nameof(OAuth2TokenProvider)));
});

builder.Services.AddHttpClient<KeywordPlannerClient>(http =>
{
    http.Timeout = TimeSpan.FromSeconds(30);
});

builder.Services.AddTransient<KeywordPlannerClient>(sp =>
{
    var factory = sp.GetRequiredService<IHttpClientFactory>();
    var tokenProvider = sp.GetRequiredService<OAuth2TokenProvider>();
    return new KeywordPlannerClient(
        developerToken, customerId, tokenProvider,
        factory.CreateClient(nameof(KeywordPlannerClient)));
});

builder.Services
    .AddMcpServer()
    .WithStdioServerTransport()
    .WithTools<KeywordPlannerTool>();

var host = builder.Build();
await host.RunAsync().ConfigureAwait(false);
return 0;
