using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace KeywordPlannerMcp.KeywordPlanner;

/// <summary>Obtains and caches OAuth2 Bearer tokens for the Google Ads API.</summary>
internal sealed class OAuth2TokenProvider(
    string clientId,
    string clientSecret,
    string refreshToken,
    HttpClient httpClient)
{
    private const string TokenEndpoint = "https://oauth2.googleapis.com/token";
    private const int ExpiryBufferSeconds = 60;

    private string? _cachedToken;
    private DateTimeOffset _tokenExpiry = DateTimeOffset.MinValue;

    internal async Task<string> GetAccessTokenAsync(CancellationToken cancellationToken = default)
    {
        if (_cachedToken is not null && DateTimeOffset.UtcNow < _tokenExpiry)
            return _cachedToken;

        var requestBody = new FormUrlEncodedContent(
        [
            new KeyValuePair<string, string>("client_id", clientId),
            new KeyValuePair<string, string>("client_secret", clientSecret),
            new KeyValuePair<string, string>("refresh_token", refreshToken),
            new KeyValuePair<string, string>("grant_type", "refresh_token"),
        ]);

        var response = await httpClient.PostAsync(TokenEndpoint, requestBody, cancellationToken);

        if (!response.IsSuccessStatusCode)
        {
            var body = await response.Content.ReadAsStringAsync(cancellationToken);
            throw new InvalidOperationException(
                $"OAuth2 token exchange failed ({response.StatusCode}): {body}");
        }

        var stream = await response.Content.ReadAsStreamAsync(cancellationToken);
        var tokenResponse = await JsonSerializer.DeserializeAsync(
            stream, KwpJsonContext.Default.TokenResponse, cancellationToken);

        if (tokenResponse is null || string.IsNullOrWhiteSpace(tokenResponse.AccessToken))
            throw new InvalidOperationException("OAuth2 response did not include an access_token.");

        _cachedToken = tokenResponse.AccessToken;
        _tokenExpiry = DateTimeOffset.UtcNow.AddSeconds(tokenResponse.ExpiresIn - ExpiryBufferSeconds);
        return _cachedToken;
    }

    /// <summary>Applies a fresh Bearer token to the given request.</summary>
    internal async Task AuthorizeAsync(HttpRequestMessage request, CancellationToken cancellationToken = default)
    {
        var token = await GetAccessTokenAsync(cancellationToken);
        request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token);
    }

    internal static StringContent JsonContent(string json) =>
        new(json, Encoding.UTF8, "application/json");
}
