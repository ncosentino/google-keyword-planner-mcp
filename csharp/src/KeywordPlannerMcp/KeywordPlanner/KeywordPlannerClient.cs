using System.Globalization;
using System.Runtime.ExceptionServices;
using System.Text.Json;
using System.Text.Json.Serialization.Metadata;

namespace KeywordPlannerMcp.KeywordPlanner;

internal sealed partial class KeywordPlannerClient(
    string developerToken,
    string customerId,
    string? loginCustomerId,
    OAuth2TokenProvider tokenProvider,
    HttpClient httpClient,
    string? defaultLanguage = null,
    IReadOnlyList<string>? defaultGeoTargets = null,
    TimeProvider? timeProvider = null,
    IPlanningPacer? planningPacer = null) : IDisposable
{
    private readonly string _baseUrl = $"https://googleads.googleapis.com/v25/customers/{customerId}";
    private readonly TimeProvider _clock = timeProvider ?? TimeProvider.System;
    private readonly IPlanningPacer _pacer = planningPacer ?? new PlanningPacer(timeProvider ?? TimeProvider.System);
    private readonly bool _ownsPacer = planningPacer is null;

    internal async Task<KeywordIdeasResponse> GenerateKeywordIdeasAsync(
        IReadOnlyList<string>? seedKeywords,
        string? url,
        string? language,
        CancellationToken cancellationToken = default,
        IReadOnlyList<string>? geoTargetConstants = null,
        string? keywordPlanNetwork = null,
        IdeaOptions? options = null)
    {
        options ??= new IdeaOptions();
        if (options.MinMonthlySearches < 0)
            throw new ArgumentException("minMonthlySearches must not be negative.");
        if (options.Limit < -1)
            throw new ArgumentException("limit must be -1, zero, or positive.");
        var targeting = await ResolveTargetingAsync(language, geoTargetConstants, cancellationToken);
        var hasSeed = seedKeywords is { Count: > 0 };
        var hasUrl = !string.IsNullOrWhiteSpace(url);
        if (!hasSeed && !hasUrl)
            throw new ArgumentException("At least one seed keyword or URL is required.");
        var request = new GenerateKeywordIdeasRequest
        {
            Language = targeting.Language,
            GeoTargetConstants = targeting.Geography,
            KeywordPlanNetwork = ResolveNetwork(keywordPlanNetwork)
        };
        if (hasSeed && hasUrl)
            request.KeywordAndUrlSeed = new KeywordAndUrlSeed { Keywords = seedKeywords!, Url = url! };
        else if (hasSeed)
            request.KeywordSeed = new KeywordSeed { Keywords = seedKeywords! };
        else
            request.UrlSeed = new UrlSeed { Url = url! };

        var results = new List<KeywordIdeaResult>();
        var seenTokens = new HashSet<string>(StringComparer.Ordinal);
        while (true)
        {
            var page = await PostAsync("generateKeywordIdeas", request,
                KwpJsonContext.Default.GenerateKeywordIdeasRequest,
                KwpJsonContext.Default.GenerateKeywordIdeasResponse, cancellationToken);
            results.AddRange(page.Results ?? []);
            if (string.IsNullOrEmpty(page.NextPageToken)) break;
            if (!seenTokens.Add(page.NextPageToken))
                throw new InvalidOperationException("Keyword ideas API repeated a pagination token.");
            request.PageToken = page.NextPageToken;
        }
        var ideas = results.Select(r => new KeywordIdea(
            r.Text, ParseLong(r.KeywordIdeaMetrics.AvgMonthlySearches),
            NormalizeCompetition(r.KeywordIdeaMetrics.Competition),
            ParseLong(r.KeywordIdeaMetrics.LowTopOfPageBidMicros),
            ParseLong(r.KeywordIdeaMetrics.HighTopOfPageBidMicros),
            checked((int)ParseLong(r.KeywordIdeaMetrics.CompetitionIndex)),
            options.IncludeMonthlyVolumes ? ToMonthlyVolumes(r.KeywordIdeaMetrics.MonthlySearchVolumes) : null))
            .Where(i => i.AvgMonthlySearches >= options.MinMonthlySearches)
            .OrderByDescending(i => i.AvgMonthlySearches);
        var selected = (options.Limit > 0 ? ideas.Take(options.Limit) : ideas).ToList();
        return new KeywordIdeasResponse(seedKeywords, url, selected, selected.Count, results.Count,
            targeting.Language, targeting.Geography);
    }

    internal async Task<HistoricalMetricsResponse> GetHistoricalMetricsAsync(
        IReadOnlyList<string> keywords,
        CancellationToken cancellationToken = default,
        string? language = null,
        IReadOnlyList<string>? geoTargetConstants = null,
        string? keywordPlanNetwork = null)
    {
        if (keywords.Count == 0)
            throw new ArgumentException("At least one keyword is required.");
        var targeting = await ResolveTargetingAsync(language, geoTargetConstants, cancellationToken);
        var request = new GenerateHistoricalMetricsRequest
        {
            Keywords = keywords, Language = targeting.Language,
            GeoTargetConstants = targeting.Geography,
            KeywordPlanNetwork = ResolveNetwork(keywordPlanNetwork)
        };
        var raw = await PostAsync("generateKeywordHistoricalMetrics", request,
            KwpJsonContext.Default.GenerateHistoricalMetricsRequest,
            KwpJsonContext.Default.GenerateHistoricalMetricsResponse, cancellationToken);
        var metrics = (raw.Results ?? []).Select(m => new KeywordMetrics(
            m.Text, ParseLong(m.KeywordMetrics.AvgMonthlySearches),
            NormalizeCompetition(m.KeywordMetrics.Competition),
            checked((int)ParseLong(m.KeywordMetrics.CompetitionIndex)),
            ParseLong(m.KeywordMetrics.LowTopOfPageBidMicros),
            ParseLong(m.KeywordMetrics.HighTopOfPageBidMicros),
            (m.KeywordMetrics.MonthlySearchVolumes ?? []).Select(v => new MonthlyVolume(
                checked((int)ParseLong(v.Year)), MonthNameToInt(v.Month),
                ParseLong(v.MonthlySearches))).ToList())).ToList();
        return new HistoricalMetricsResponse(metrics, metrics.Count);
    }

    internal async Task<ForecastResponse> GetKeywordForecastAsync(
        IReadOnlyList<string> keywords,
        long maxCpcMicros,
        int forecastDays,
        CancellationToken cancellationToken = default,
        string? language = null,
        IReadOnlyList<string>? geoTargetConstants = null,
        string? matchType = null,
        string? currencyCode = null,
        bool includeKeywordBreakdown = false)
    {
        if (keywords.Count == 0)
            throw new ArgumentException("At least one keyword is required.");
        if (forecastDays <= 0) forecastDays = 30;
        if (forecastDays > 364)
            throw new ArgumentException("forecastDays must not exceed 364.");
        if (maxCpcMicros <= 0) maxCpcMicros = 1_000_000;
        var normalizedMatchType = string.IsNullOrWhiteSpace(matchType) ? "BROAD" : matchType.Trim().ToUpperInvariant();
        if (normalizedMatchType is not ("BROAD" or "PHRASE" or "EXACT"))
            throw new ArgumentException("matchType must be BROAD, PHRASE, or EXACT.");
        var targeting = await ResolveTargetingAsync(language, geoTargetConstants, cancellationToken);
        var start = DateOnly.FromDateTime(_clock.GetUtcNow().UtcDateTime).AddDays(2);
        var period = new ForecastPeriod
        {
            StartDate = start.ToString("yyyy-MM-dd", CultureInfo.InvariantCulture),
            EndDate = start.AddDays(forecastDays - 1).ToString("yyyy-MM-dd", CultureInfo.InvariantCulture)
        };
        currencyCode = string.IsNullOrEmpty(currencyCode) ? null : currencyCode;
        async Task<KeywordForecastMetrics> ForecastAsync(
            IReadOnlyList<string> scenario, string? text, CancellationToken token)
        {
            var request = new GenerateForecastMetricsRequest
            {
                ForecastPeriod = period,
                CurrencyCode = currencyCode,
                Campaign = new CampaignForecast
                {
                    LanguageConstants = targeting.Language is null ? null : [targeting.Language],
                    GeoTargetConstants = targeting.Geography,
                    BiddingStrategy = new BiddingStrategy
                    {
                        ManualCpcBiddingStrategy = new ManualCpcBiddingStrategy
                        {
                            MaxCpcBidMicros = maxCpcMicros.ToString(CultureInfo.InvariantCulture)
                        }
                    },
                    AdGroups = [new AdGroupForecast
                    {
                        Keywords = scenario.Select(k => new ForecastKeyword { Text = k, MatchType = normalizedMatchType }).ToList()
                    }]
                }
            };
            var raw = await PostAsync("generateKeywordForecastMetrics", request,
                KwpJsonContext.Default.GenerateForecastMetricsRequest,
                KwpJsonContext.Default.GenerateForecastMetricsResponse, token);
            var metrics = raw.CampaignForecastMetrics ??
                throw new InvalidOperationException("Forecast API response did not include campaignForecastMetrics.");
            return new KeywordForecastMetrics(text, metrics.Clicks, metrics.CostMicros, metrics.AverageCpcMicros);
        }

        var scenarios = includeKeywordBreakdown ? new KeywordForecastMetrics[keywords.Count] : [];
        if (includeKeywordBreakdown)
        {
            using var siblings = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            ExceptionDispatchInfo? firstError = null;
            try
            {
                await Parallel.ForEachAsync(Enumerable.Range(0, keywords.Count),
                    new ParallelOptions { MaxDegreeOfParallelism = 5, CancellationToken = siblings.Token },
                    async (index, token) =>
                    {
                        try { scenarios[index] = await ForecastAsync([keywords[index]], keywords[index], token); }
                        catch (Exception error)
                        {
                            Interlocked.CompareExchange(ref firstError, ExceptionDispatchInfo.Capture(error), null);
                            siblings.Cancel();
                            throw;
                        }
                    });
            }
            catch
            {
                cancellationToken.ThrowIfCancellationRequested();
                firstError?.Throw();
                throw;
            }
        }
        cancellationToken.ThrowIfCancellationRequested();
        var total = scenarios.Length == 1
            ? scenarios[0] with { Text = null }
            : await ForecastAsync(keywords, null, cancellationToken);
        cancellationToken.ThrowIfCancellationRequested();
        return new ForecastResponse(scenarios, total, forecastDays, maxCpcMicros,
            period.StartDate, period.EndDate, targeting.Language, targeting.Geography, currencyCode, normalizedMatchType);
    }

    private async Task<TResponse> PostAsync<TRequest, TResponse>(
        string endpoint, TRequest payload, JsonTypeInfo<TRequest> requestType,
        JsonTypeInfo<TResponse> responseType, CancellationToken cancellationToken)
        where TResponse : class
    {
        cancellationToken.ThrowIfCancellationRequested();
        var planning = endpoint is "generateKeywordIdeas" or "generateKeywordHistoricalMetrics" or "generateKeywordForecastMetrics";
        var address = planning ? $"{_baseUrl}:{endpoint}" : endpoint == "googleAds:search"
            ? $"{_baseUrl}/{endpoint}" : $"https://googleads.googleapis.com/v25/{endpoint}";
        using var request = new HttpRequestMessage(HttpMethod.Post, address);
        request.Content = OAuth2TokenProvider.JsonContent(JsonSerializer.Serialize(payload, requestType));
        await tokenProvider.AuthorizeAsync(request, cancellationToken);
        if (!string.IsNullOrEmpty(developerToken))
            request.Headers.Add("developer-token", developerToken);
        if (!string.IsNullOrWhiteSpace(loginCustomerId))
            request.Headers.Add("login-customer-id", loginCustomerId);
        if (planning) await _pacer.WaitAsync(cancellationToken);
        using var response = await httpClient.SendAsync(request, cancellationToken);
        var body = await response.Content.ReadAsStringAsync(cancellationToken);
        cancellationToken.ThrowIfCancellationRequested();
        if (!response.IsSuccessStatusCode)
            throw new InvalidOperationException($"{endpoint} API error ({response.StatusCode}): {body}");
        return JsonSerializer.Deserialize(body, responseType)
            ?? throw new InvalidOperationException($"{endpoint} API returned an empty response.");
    }

    private async Task<(string? Language, IReadOnlyList<string>? Geography)> ResolveTargetingAsync(
        string? language, IReadOnlyList<string>? geography, CancellationToken cancellationToken)
    {
        language = string.IsNullOrEmpty(language) ? defaultLanguage : language;
        if (language is not null && language.Length != 0 && string.IsNullOrWhiteSpace(language))
            throw new ArgumentException("language must not be blank; use 'all' for all languages.");
        language = language?.Trim();
        if (string.IsNullOrEmpty(language) || string.Equals(language, "all", StringComparison.OrdinalIgnoreCase))
            language = null;
        else
            language = await ResolveConstantAsync(language, true, cancellationToken);
        geography ??= defaultGeoTargets;
        var targets = new List<string>();
        foreach (var target in geography ?? [])
        {
            if (string.IsNullOrWhiteSpace(target))
                throw new ArgumentException("geoTargetConstants entries must not be blank; use an empty list for worldwide targeting.");
            var trimmed = await ResolveConstantAsync(target.Trim(), false, cancellationToken);
            if (!targets.Contains(trimmed, StringComparer.Ordinal)) targets.Add(trimmed);
        }
        if (targets.Count > 10)
            throw new ArgumentException("At most 10 unique geoTargetConstants are supported.");
        return (language, targets.Count == 0 ? null : targets);
    }

    private static string? ResolveNetwork(string? network)
    {
        if (string.IsNullOrEmpty(network)) return null;
        network = network.Trim().ToUpperInvariant();
        if (network is not ("GOOGLE_SEARCH" or "GOOGLE_SEARCH_AND_PARTNERS"))
            throw new ArgumentException("keywordPlanNetwork must be GOOGLE_SEARCH or GOOGLE_SEARCH_AND_PARTNERS.");
        return network;
    }

    private static long ParseLong(string? value) =>
        long.TryParse(value, NumberStyles.Integer, CultureInfo.InvariantCulture, out var n) ? n : 0L;

    private static List<MonthlyVolume> ToMonthlyVolumes(List<RawMonthlyVolume>? volumes) =>
        (volumes ?? []).Select(v => new MonthlyVolume(checked((int)ParseLong(v.Year)),
            MonthNameToInt(v.Month), ParseLong(v.MonthlySearches))).ToList();

    private static string NormalizeCompetition(string raw) =>
        raw is "COMPETITION_UNSPECIFIED" or "" ? "UNKNOWN" : raw;

    private static int MonthNameToInt(string month) => month switch
    {
        "JANUARY" => 1, "FEBRUARY" => 2, "MARCH" => 3, "APRIL" => 4,
        "MAY" => 5, "JUNE" => 6, "JULY" => 7, "AUGUST" => 8,
        "SEPTEMBER" => 9, "OCTOBER" => 10, "NOVEMBER" => 11, "DECEMBER" => 12, _ => 0
    };

    public void Dispose()
    {
        _languageGate.Dispose();
        _countryGate.Dispose();
        if (_ownsPacer && _pacer is IDisposable disposable)
            disposable.Dispose();
    }
}

internal interface IPlanningPacer
{
    Task WaitAsync(CancellationToken cancellationToken);
}

internal sealed class PlanningPacer(TimeProvider clock) : IPlanningPacer, IDisposable
{
    private readonly SemaphoreSlim _gate = new(1, 1);
    private long? _lastRequest;

    public async Task WaitAsync(CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken);
        try
        {
            if (_lastRequest is { } last)
            {
                var delay = TimeSpan.FromSeconds(1) - clock.GetElapsedTime(last);
                if (delay > TimeSpan.Zero)
                    await Task.Delay(delay, clock, cancellationToken);
            }
            cancellationToken.ThrowIfCancellationRequested();
            _lastRequest = clock.GetTimestamp();
        }
        finally { _gate.Release(); }
    }

    public void Dispose() => _gate.Dispose();
}
