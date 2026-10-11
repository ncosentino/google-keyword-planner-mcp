using System.ComponentModel;
using System.Text.Json;
using KeywordPlannerMcp.KeywordPlanner;
using ModelContextProtocol.Server;

namespace KeywordPlannerMcp.Tools;

/// <summary>MCP tools that expose Google Ads Keyword Planner functionality.</summary>
[McpServerToolType]
internal sealed class KeywordPlannerTool(KeywordPlannerClient client)
{
    [McpServerTool(Name = "generate_keyword_ideas")]
    [Description("Generate keyword ideas from seed keywords and/or a URL using Google Ads Keyword Planner. Returns related keywords with average monthly search volume, competition level, and CPC estimates.")]
    internal async Task<string> GenerateKeywordIdeasAsync(
        [Description("Comma-separated seed keywords (e.g. 'C# tutorial, dotnet performance').")] string? seedKeywords = null,
        [Description("A URL to generate ideas from (e.g. 'https://devleader.ca').")] string? url = null,
        [Description("Language resource name, bare ID, ISO code, or English name. Omit to use the configured default; 'all' removes the restriction.")] string? language = null,
        CancellationToken cancellationToken = default,
        [Description("Comma-separated geo resource names, IDs, country ISO codes, or English names. Omit to use defaults; an empty string means worldwide.")] string? geoTargetConstants = null,
        [Description("GOOGLE_SEARCH or GOOGLE_SEARCH_AND_PARTNERS. Omit to use the API default.")] string? keywordPlanNetwork = null,
        [Description("Maximum ideas, sorted by volume. Defaults to 50; zero also means 50, -1 returns all.")] int limit = 50,
        [Description("Minimum average monthly searches, nonnegative.")] long minMonthlySearches = 0,
        [Description("Include per-month search volumes for seasonality analysis. Defaults to false.")] bool includeMonthlyVolumes = false)
    {
        var seeds = string.IsNullOrWhiteSpace(seedKeywords)
            ? null
            : seedKeywords.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);

        if (seeds is null && string.IsNullOrWhiteSpace(url))
        {
            var validationError = new ErrorResult("At least one of seedKeywords or url must be provided.");
            return JsonSerializer.Serialize(validationError, KwpJsonContext.Default.ErrorResult);
        }

        try
        {
            var result = await client.GenerateKeywordIdeasAsync(seeds, url, language, cancellationToken,
                ParseGeography(geoTargetConstants), keywordPlanNetwork,
                new IdeaOptions(limit == 0 ? 50 : limit, minMonthlySearches, includeMonthlyVolumes))
                .ConfigureAwait(false);
            return JsonSerializer.Serialize(result, KwpJsonContext.Default.KeywordIdeasResponse);
        }
        catch (Exception ex) when (ex is not OperationCanceledException)
        {
            var err = new ErrorResult($"Error generating keyword ideas: {ex.Message}");
            return JsonSerializer.Serialize(err, KwpJsonContext.Default.ErrorResult);
        }
    }

    [McpServerTool(Name = "search_geo_targets")]
    [Description("Find country, region, and city geo target resource names for keyword targeting.")]
    internal async Task<string> SearchGeoTargetsAsync(
        [Description("Comma-separated location names, e.g. 'Vienna, Salzburg'.")] string names,
        [Description("Optional ISO country code restricting suggestions.")] string? countryCode = null,
        [Description("Locale for returned names. Defaults to en.")] string? locale = null,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var result = await client.SearchGeoTargetsAsync(
                names.Split(',', StringSplitOptions.TrimEntries), countryCode, locale, cancellationToken);
            return JsonSerializer.Serialize(result, KwpJsonContext.Default.GeoTargetSearchResponse);
        }
        catch (Exception ex) when (ex is not OperationCanceledException)
        {
            return JsonSerializer.Serialize(new ErrorResult($"Error searching geo targets: {ex.Message}"),
                KwpJsonContext.Default.ErrorResult);
        }
    }

    [McpServerTool(Name = "get_historical_metrics")]
    [Description("Get historical search volume and competition metrics for a list of keywords using Google Ads Keyword Planner.")]
    internal async Task<string> GetHistoricalMetricsAsync(
        [Description("Comma-separated list of keywords to look up (e.g. 'dependency injection, SOLID principles').")] string keywords,
        CancellationToken cancellationToken = default,
        [Description("Language resource name, ID, ISO code, or English name. Omit to use defaults; 'all' means unrestricted.")] string? language = null,
        [Description("Comma-separated geo resource names, IDs, country ISO codes, or English names. Omit to use defaults; an empty string means worldwide.")] string? geoTargetConstants = null,
        [Description("GOOGLE_SEARCH or GOOGLE_SEARCH_AND_PARTNERS. Omit to use the API default.")] string? keywordPlanNetwork = null)
    {
        var keywordList = keywords.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);

        try
        {
            var result = await client.GetHistoricalMetricsAsync(keywordList, cancellationToken,
                language, ParseGeography(geoTargetConstants), keywordPlanNetwork)
                .ConfigureAwait(false);
            return JsonSerializer.Serialize(result, KwpJsonContext.Default.HistoricalMetricsResponse);
        }
        catch (Exception ex) when (ex is not OperationCanceledException)
        {
            var err = new ErrorResult($"Error getting historical metrics: {ex.Message}");
            return JsonSerializer.Serialize(err, KwpJsonContext.Default.ErrorResult);
        }
    }

    [McpServerTool(Name = "get_keyword_forecast")]
    [Description("Get projected campaign clicks, costMicros, and averageCpcMicros. Optional per-keyword results are independent scenarios, not additive. maxCpcMicros is in micros (1,000,000 = one currency unit). forecastDays defaults to 30.")]
    internal async Task<string> GetKeywordForecastAsync(
        [Description("Comma-separated list of keywords to forecast.")] string keywords,
        [Description("Maximum CPC bid in micros (1,000,000 = one requested or account currency unit). Defaults to 1,000,000.")] long maxCpcMicros = 1_000_000,
        [Description("Number of days to forecast. Defaults to 30.")] int forecastDays = 30,
        CancellationToken cancellationToken = default,
        [Description("Language resource name, ID, ISO code, or English name. Omit to use defaults; 'all' means unrestricted.")] string? language = null,
        [Description("Comma-separated geo resource names, IDs, country ISO codes, or English names. Omit to use defaults; an empty string means worldwide.")] string? geoTargetConstants = null,
        [Description("BROAD, PHRASE, or EXACT. Defaults to BROAD.")] string? matchType = null,
        [Description("Optional currency code, echoed when supplied.")] string? currencyCode = null,
        [Description("Request independent per-keyword scenarios in addition to the combined total. Requires extra API calls; scenarios are not additive.")] bool includeKeywordBreakdown = false)
    {
        var keywordList = keywords.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);

        try
        {
            var result = await client.GetKeywordForecastAsync(keywordList, maxCpcMicros, forecastDays,
                cancellationToken, language, ParseGeography(geoTargetConstants), matchType,
                currencyCode, includeKeywordBreakdown)
                .ConfigureAwait(false);
            return JsonSerializer.Serialize(result, KwpJsonContext.Default.ForecastResponse);
        }
        catch (Exception ex) when (ex is not OperationCanceledException)
        {
            var err = new ErrorResult($"Error getting keyword forecast: {ex.Message}");
            return JsonSerializer.Serialize(err, KwpJsonContext.Default.ErrorResult);
        }

    }

    internal static IReadOnlyList<string>? ParseGeography(string? value) =>
        value is null ? null : value.Length == 0 ? [] : value.Split(',', StringSplitOptions.TrimEntries);
}
