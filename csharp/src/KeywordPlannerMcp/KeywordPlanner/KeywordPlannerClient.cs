using System.Text.Json;

namespace KeywordPlannerMcp.KeywordPlanner;

/// <summary>Client for the Google Ads Keyword Planner REST API.</summary>
internal sealed class KeywordPlannerClient(
    string developerToken,
    string customerId,
    string? loginCustomerId,
    OAuth2TokenProvider tokenProvider,
    HttpClient httpClient)
{
    private readonly string _baseUrl = $"https://googleads.googleapis.com/v23/customers/{customerId}";

    /// <summary>Generates keyword ideas from seed keywords and/or a URL.</summary>
    internal async Task<KeywordIdeasResponse> GenerateKeywordIdeasAsync(
        IReadOnlyList<string>? seedKeywords,
        string? url,
        string? language,
        CancellationToken cancellationToken = default)
    {
        var request = BuildKeywordIdeasRequest(seedKeywords, url, language);
        var json = JsonSerializer.Serialize(request, KwpJsonContext.Default.GenerateKeywordIdeasRequest);

        using var httpRequest = new HttpRequestMessage(
            HttpMethod.Post, $"{_baseUrl}:generateKeywordIdeas");
        await tokenProvider.AuthorizeAsync(httpRequest, cancellationToken);
        AddAuthHeaders(httpRequest);
        httpRequest.Content = OAuth2TokenProvider.JsonContent(json);

        using var response = await httpClient.SendAsync(httpRequest, cancellationToken);
        var body = await response.Content.ReadAsStringAsync(cancellationToken);

        if (!response.IsSuccessStatusCode)
            throw new InvalidOperationException($"Keyword ideas API error ({response.StatusCode}): {body}");

        var raw = JsonSerializer.Deserialize(body, KwpJsonContext.Default.GenerateKeywordIdeasResponse);
        var ideas = (raw?.Results ?? [])
            .Select(r => new KeywordIdea(
                r.Text,
                ParseLong(r.KeywordIdeaMetrics.AvgMonthlySearches),
                NormalizeCompetition(r.KeywordIdeaMetrics.Competition),
                ParseLong(r.KeywordIdeaMetrics.LowTopOfPageBidMicros),
                ParseLong(r.KeywordIdeaMetrics.HighTopOfPageBidMicros)))
            .ToList();

        return new KeywordIdeasResponse(seedKeywords, url, ideas, ideas.Count);
    }

    /// <summary>Gets historical search volume and competition metrics for specific keywords.</summary>
    internal async Task<HistoricalMetricsResponse> GetHistoricalMetricsAsync(
        IReadOnlyList<string> keywords,
        CancellationToken cancellationToken = default)
    {
        var request = new GenerateHistoricalMetricsRequest { Keywords = keywords };
        var json = JsonSerializer.Serialize(
            request, KwpJsonContext.Default.GenerateHistoricalMetricsRequest);

        using var httpRequest = new HttpRequestMessage(
            HttpMethod.Post, $"{_baseUrl}:generateKeywordHistoricalMetrics");
        await tokenProvider.AuthorizeAsync(httpRequest, cancellationToken);
        AddAuthHeaders(httpRequest);
        httpRequest.Content = OAuth2TokenProvider.JsonContent(json);

        using var response = await httpClient.SendAsync(httpRequest, cancellationToken);
        var body = await response.Content.ReadAsStringAsync(cancellationToken);

        if (!response.IsSuccessStatusCode)
            throw new InvalidOperationException($"Historical metrics API error ({response.StatusCode}): {body}");

        var raw = JsonSerializer.Deserialize(
            body, KwpJsonContext.Default.GenerateHistoricalMetricsResponse);

        var metrics = (raw?.Metrics ?? [])
            .Select(m => new KeywordMetrics(
                m.Text,
                ParseLong(m.KeywordMetrics.AvgMonthlySearches),
                NormalizeCompetition(m.KeywordMetrics.Competition),
                m.KeywordMetrics.CompetitionIndex,
                ParseLong(m.KeywordMetrics.LowTopOfPageBidMicros),
                ParseLong(m.KeywordMetrics.HighTopOfPageBidMicros),
                (m.KeywordMetrics.MonthlySearchVolumes ?? [])
                    .Select(v => new MonthlyVolume(
                        v.Year,
                        MonthNameToInt(v.Month),
                        ParseLong(v.MonthlySearches)))
                    .ToList()))
            .ToList();

        return new HistoricalMetricsResponse(metrics, metrics.Count);
    }

    /// <summary>Gets projected performance metrics for a set of keywords at a given max CPC.</summary>
    internal async Task<ForecastResponse> GetKeywordForecastAsync(
        IReadOnlyList<string> keywords,
        long maxCpcMicros,
        int forecastDays,
        CancellationToken cancellationToken = default)
    {
        var request = BuildForecastRequest(keywords, maxCpcMicros, forecastDays);
        var json = JsonSerializer.Serialize(
            request, KwpJsonContext.Default.GenerateForecastMetricsRequest);

        using var httpRequest = new HttpRequestMessage(
            HttpMethod.Post, $"{_baseUrl}:generateKeywordForecastMetrics");
        await tokenProvider.AuthorizeAsync(httpRequest, cancellationToken);
        AddAuthHeaders(httpRequest);
        httpRequest.Content = OAuth2TokenProvider.JsonContent(json);

        using var response = await httpClient.SendAsync(httpRequest, cancellationToken);
        var body = await response.Content.ReadAsStringAsync(cancellationToken);

        if (!response.IsSuccessStatusCode)
            throw new InvalidOperationException($"Forecast API error ({response.StatusCode}): {body}");

        var raw = JsonSerializer.Deserialize(
            body, KwpJsonContext.Default.GenerateForecastMetricsResponse);

        var forecastKeywords = (raw?.AdGroupForecastMetrics ?? [])
            .SelectMany(ag => ag.KeywordForecastMetrics ?? [])
            .Select(kf => new KeywordForecastMetrics(
                kf.Keyword.Text,
                kf.Metrics.Impressions,
                kf.Metrics.Clicks,
                kf.Metrics.CostMicros,
                kf.Metrics.Ctr))
            .ToList();

        return new ForecastResponse(forecastKeywords, forecastDays, maxCpcMicros);
    }

    private static GenerateKeywordIdeasRequest BuildKeywordIdeasRequest(
        IReadOnlyList<string>? seedKeywords,
        string? url,
        string? language)
    {
        var hasSeed = seedKeywords is { Count: > 0 };
        var hasUrl = !string.IsNullOrWhiteSpace(url);

        var request = new GenerateKeywordIdeasRequest { Language = language };
        if (hasSeed && hasUrl)
        {
            request.KeywordAndUrlSeed = new KeywordAndUrlSeed
            {
                Keywords = seedKeywords!,
                Url = url!
            };
        }
        else if (hasSeed)
        {
            request.KeywordSeed = new KeywordSeed { Keywords = seedKeywords! };
        }
        else if (hasUrl)
        {
            request.UrlSeed = new UrlSeed { Url = url! };
        }

        return request;
    }

    private static GenerateForecastMetricsRequest BuildForecastRequest(
        IReadOnlyList<string> keywords,
        long maxCpcMicros,
        int forecastDays)
    {
        var start = DateTime.UtcNow.ToString("yyyy-MM-dd");
        var end = DateTime.UtcNow.AddDays(forecastDays).ToString("yyyy-MM-dd");

        return new GenerateForecastMetricsRequest
        {
            CampaignForecastSpec = new CampaignForecastSpec
            {
                StartDate = start,
                EndDate = end,
                BiddingStrategy = new BiddingStrategy
                {
                    ManualCpcBiddingStrategy = new ManualCpcBiddingStrategy
                    {
                        MaxCpcBidMicros = maxCpcMicros.ToString()
                    }
                },
                AdGroups =
                [
                    new AdGroupForecast
                    {
                        BiddableKeywords = keywords
                            .Select(k => new AdGroupForecastKeyword
                            {
                                Keyword = new ForecastKeyword { Text = k, MatchType = "BROAD" }
                            })
                            .ToList()
                    }
                ]
            }
        };
    }

    private static long ParseLong(string? value) =>
        long.TryParse(value, out var n) ? n : 0L;

    private void AddAuthHeaders(HttpRequestMessage request)
    {
        request.Headers.Add("developer-token", developerToken);
        if (!string.IsNullOrWhiteSpace(loginCustomerId))
            request.Headers.Add("login-customer-id", loginCustomerId);
    }

    private static string NormalizeCompetition(string raw) =>
        raw switch
        {
            "COMPETITION_UNSPECIFIED" => "UNKNOWN",
            "" => "UNKNOWN",
            _ => raw
        };

    private static int MonthNameToInt(string month) =>
        month switch
        {
            "JANUARY" => 1,
            "FEBRUARY" => 2,
            "MARCH" => 3,
            "APRIL" => 4,
            "MAY" => 5,
            "JUNE" => 6,
            "JULY" => 7,
            "AUGUST" => 8,
            "SEPTEMBER" => 9,
            "OCTOBER" => 10,
            "NOVEMBER" => 11,
            "DECEMBER" => 12,
            _ => 0
        };
}
