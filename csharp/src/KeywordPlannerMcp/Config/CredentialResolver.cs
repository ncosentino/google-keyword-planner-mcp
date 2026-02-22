using System.Runtime.CompilerServices;

namespace KeywordPlannerMcp.Config;

/// <summary>Resolves Google Ads API credentials from multiple sources.</summary>
/// <remarks>
/// Each credential is resolved independently using priority order:
/// CLI flag > environment variable > .env file.
/// </remarks>
internal static class CredentialResolver
{
    internal const string EnvDeveloperToken = "GOOGLE_ADS_DEVELOPER_TOKEN";
    internal const string EnvClientId = "GOOGLE_ADS_CLIENT_ID";
    internal const string EnvClientSecret = "GOOGLE_ADS_CLIENT_SECRET";
    internal const string EnvRefreshToken = "GOOGLE_ADS_REFRESH_TOKEN";
    internal const string EnvCustomerId = "GOOGLE_ADS_CUSTOMER_ID";
    internal const string EnvLoginCustomerId = "GOOGLE_ADS_LOGIN_CUSTOMER_ID";
    private const string DotEnvFile = ".env";

    internal static string? ResolveCredential(string? flagValue, string envVarName)
    {
        if (!string.IsNullOrWhiteSpace(flagValue))
            return flagValue;

        var envValue = Environment.GetEnvironmentVariable(envVarName);
        if (!string.IsNullOrWhiteSpace(envValue))
            return envValue;

        return ReadFromDotEnv(envVarName);
    }

    [MethodImpl(MethodImplOptions.NoInlining)]
    private static string? ReadFromDotEnv(string envVarName)
    {
        if (!File.Exists(DotEnvFile))
            return null;

        var prefix = envVarName + "=";
        foreach (var line in File.ReadLines(DotEnvFile))
        {
            var trimmed = line.Trim();
            if (trimmed.StartsWith('#') || trimmed.Length == 0)
                continue;

            if (trimmed.StartsWith(prefix, StringComparison.Ordinal))
            {
                var value = trimmed[prefix.Length..].Trim('"', '\'');
                return string.IsNullOrWhiteSpace(value) ? null : value;
            }
        }

        return null;
    }

    /// <summary>Strips dashes from a customer ID (e.g. "123-456-7890" -> "1234567890").</summary>
    internal static string? NormalizeCustomerId(string? id) =>
        id?.Replace("-", string.Empty, StringComparison.Ordinal);
}
