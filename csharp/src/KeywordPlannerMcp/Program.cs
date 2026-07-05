using System.Globalization;
using KeywordPlannerMcp;
using KeywordPlannerMcp.Config;
using KeywordPlannerMcp.Tools;
using Microsoft.Extensions.Hosting;
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

// --transport has no environment-variable fallback, matching the Go implementation.
string transport = args.SkipWhile(a => a != "--transport").Skip(1).FirstOrDefault() ?? "stdio";

if (transport == "http")
{
    var port = Environment.GetEnvironmentVariable("PORT") is { Length: > 0 } portEnv ? portEnv : "8080";
    var app = Hosting.BuildHttpHost(
        args, developerToken, clientId, clientSecret, refreshToken, customerId, loginCustomerId,
        int.Parse(port, CultureInfo.InvariantCulture));
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
