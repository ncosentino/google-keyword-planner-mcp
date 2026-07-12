using KeywordPlannerMcp;
using KeywordPlannerMcp.Config;
using KeywordPlannerMcp.Tools;
using Microsoft.Extensions.Hosting;
using ModelContextProtocol.Server;

if (ServerOptions.IsHelpRequested(args))
{
    await Console.Out.WriteLineAsync(ServerOptions.Usage).ConfigureAwait(false);
    return 0;
}

ServerOptions options;
try
{
    options = ServerOptions.Parse(args);
}
catch (ArgumentException exception)
{
    await Console.Error.WriteLineAsync($"ERROR: {exception.Message}").ConfigureAwait(false);
    return 1;
}

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

string? loginCustomerId = CredentialResolver.NormalizeCustomerId(
    CredentialResolver.ResolveCredential(
        args.SkipWhile(a => a != "--login-customer-id").Skip(1).FirstOrDefault(),
        CredentialResolver.EnvLoginCustomerId));

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

if (options.Transport == "http")
{
    var app = Hosting.BuildHttpHost(
        args,
        developerToken,
        clientId,
        clientSecret,
        refreshToken,
        customerId,
        loginCustomerId,
        options.Port,
        listenAddress: options.ListenAddress,
        shutdownToken: options.ShutdownToken);
    await app.RunAsync().ConfigureAwait(false);
    return 0;
}

var builder = Host.CreateApplicationBuilder(args);

// All logs must go to stderr to avoid corrupting the MCP STDIO stream.
Hosting.ConfigureCommonServices(builder, developerToken, clientId, clientSecret, refreshToken, customerId, loginCustomerId);

builder.Services
    .AddMcpServer()
    .WithStdioServerTransport()
    .WithTools<KeywordPlannerTool>();

var host = builder.Build();
await host.RunAsync().ConfigureAwait(false);
return 0;
