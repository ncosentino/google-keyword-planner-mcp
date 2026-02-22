using KeywordPlannerMcp.Config;
using Xunit;

namespace KeywordPlannerMcp.Tests;

public sealed class CredentialResolverTests
{
    [Fact]
    public void ResolveCredential_FlagValueTakesPrecedence()
    {
        Environment.SetEnvironmentVariable(CredentialResolver.EnvDeveloperToken, "from-env");
        try
        {
            var result = CredentialResolver.ResolveCredential("from-flag", CredentialResolver.EnvDeveloperToken);
            Assert.Equal("from-flag", result);
        }
        finally
        {
            Environment.SetEnvironmentVariable(CredentialResolver.EnvDeveloperToken, null);
        }
    }

    [Fact]
    public void ResolveCredential_EnvVarUsedWhenNoFlag()
    {
        Environment.SetEnvironmentVariable(CredentialResolver.EnvClientId, "my-client-id");
        try
        {
            var result = CredentialResolver.ResolveCredential(null, CredentialResolver.EnvClientId);
            Assert.Equal("my-client-id", result);
        }
        finally
        {
            Environment.SetEnvironmentVariable(CredentialResolver.EnvClientId, null);
        }
    }

    [Fact]
    public void ResolveCredential_ReturnsNullWhenNothingSet()
    {
        // Use an env var name that won't exist in the test environment
        var result = CredentialResolver.ResolveCredential(null, "KWP_TEST_NONEXISTENT_VAR_12345");
        Assert.Null(result);
    }

    [Theory]
    [InlineData("123-456-7890", "1234567890")]
    [InlineData("1234567890", "1234567890")]
    [InlineData(null, null)]
    public void NormalizeCustomerId_StripsDashes(string? input, string? expected)
    {
        var result = CredentialResolver.NormalizeCustomerId(input);
        Assert.Equal(expected, result);
    }

    [Fact]
    public void ResolveCredential_LoginCustomerIdEnvVar_IsResolved()
    {
        Environment.SetEnvironmentVariable(CredentialResolver.EnvLoginCustomerId, "1381404200");
        try
        {
            var result = CredentialResolver.ResolveCredential(null, CredentialResolver.EnvLoginCustomerId);
            Assert.Equal("1381404200", result);
        }
        finally
        {
            Environment.SetEnvironmentVariable(CredentialResolver.EnvLoginCustomerId, null);
        }
    }
}
