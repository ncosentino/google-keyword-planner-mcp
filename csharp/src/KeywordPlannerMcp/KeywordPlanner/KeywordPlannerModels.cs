using System.Text.Json.Serialization;

namespace KeywordPlannerMcp.KeywordPlanner;

// --- Tool result models ---

/// <summary>A keyword idea with search volume and competition data.</summary>
internal sealed record KeywordIdea(
    string Text,
    long AvgMonthlySearches,
    string Competition,
    long LowTopOfPageBidMicros,
    long HighTopOfPageBidMicros);

/// <summary>The result of generating keyword ideas.</summary>
internal sealed record KeywordIdeasResponse(
    IReadOnlyList<string>? SeedKeywords,
    string? Url,
    IReadOnlyList<KeywordIdea> Ideas,
    int Count);

/// <summary>Monthly search volume for a keyword.</summary>
internal sealed record MonthlyVolume(int Year, int Month, long MonthlySearches);

/// <summary>Historical search metrics for a keyword.</summary>
internal sealed record KeywordMetrics(
    string Text,
    long AvgMonthlySearches,
    string Competition,
    int CompetitionIndex,
    long LowTopOfPageBidMicros,
    long HighTopOfPageBidMicros,
    IReadOnlyList<MonthlyVolume> MonthlySearchVolumes);

/// <summary>The result of a historical metrics lookup.</summary>
internal sealed record HistoricalMetricsResponse(
    IReadOnlyList<KeywordMetrics> Keywords,
    int Count);

/// <summary>Projected performance metrics for a keyword.</summary>
internal sealed record KeywordForecastMetrics(
    string? Text,
    double? Clicks,
    long? CostMicros,
    long? AverageCpcMicros);

/// <summary>The result of a keyword forecast request.</summary>
internal sealed record ForecastResponse(
    IReadOnlyList<KeywordForecastMetrics> Keywords,
    KeywordForecastMetrics Total,
    int ForecastDays,
    long MaxCpcMicros,
    string StartDate,
    string EndDate,
    string? Language,
    IReadOnlyList<string>? GeoTargetConstants,
    string? CurrencyCode,
    string MatchType);

/// <summary>An error result.</summary>
internal sealed record ErrorResult(string Error);

// --- Raw Google Ads API request/response models ---

internal sealed class GenerateKeywordIdeasRequest
{
    [JsonPropertyName("language")]
    public string? Language { get; set; }

    [JsonPropertyName("geoTargetConstants")]
    public IReadOnlyList<string>? GeoTargetConstants { get; set; }

    [JsonPropertyName("keywordPlanNetwork")]
    public string? KeywordPlanNetwork { get; set; }

    [JsonPropertyName("keywordSeed")]
    public KeywordSeed? KeywordSeed { get; set; }

    [JsonPropertyName("urlSeed")]
    public UrlSeed? UrlSeed { get; set; }

    [JsonPropertyName("keywordAndUrlSeed")]
    public KeywordAndUrlSeed? KeywordAndUrlSeed { get; set; }
}

internal sealed class KeywordSeed
{
    [JsonPropertyName("keywords")]
    public IReadOnlyList<string> Keywords { get; set; } = [];
}

internal sealed class UrlSeed
{
    [JsonPropertyName("url")]
    public string Url { get; set; } = string.Empty;
}

internal sealed class KeywordAndUrlSeed
{
    [JsonPropertyName("url")]
    public string Url { get; set; } = string.Empty;

    [JsonPropertyName("keywords")]
    public IReadOnlyList<string> Keywords { get; set; } = [];
}

internal sealed class GenerateKeywordIdeasResponse
{
    [JsonPropertyName("results")]
    public List<KeywordIdeaResult>? Results { get; set; }
}

internal sealed class KeywordIdeaResult
{
    [JsonPropertyName("text")]
    public string Text { get; set; } = string.Empty;

    [JsonPropertyName("keywordIdeaMetrics")]
    public KeywordIdeaMetrics KeywordIdeaMetrics { get; set; } = new();
}

internal sealed class KeywordIdeaMetrics
{
    [JsonPropertyName("avgMonthlySearches")]
    public string? AvgMonthlySearches { get; set; }

    [JsonPropertyName("competition")]
    public string Competition { get; set; } = string.Empty;

    [JsonPropertyName("lowTopOfPageBidMicros")]
    public string? LowTopOfPageBidMicros { get; set; }

    [JsonPropertyName("highTopOfPageBidMicros")]
    public string? HighTopOfPageBidMicros { get; set; }
}

internal sealed class GenerateHistoricalMetricsRequest
{
    [JsonPropertyName("keywords")]
    public IReadOnlyList<string> Keywords { get; set; } = [];

    [JsonPropertyName("language")]
    public string? Language { get; set; }

    [JsonPropertyName("geoTargetConstants")]
    public IReadOnlyList<string>? GeoTargetConstants { get; set; }

    [JsonPropertyName("keywordPlanNetwork")]
    public string? KeywordPlanNetwork { get; set; }
}

internal sealed class GenerateHistoricalMetricsResponse
{
    [JsonPropertyName("results")]
    public List<HistoricalMetricsResult>? Results { get; set; }
}

internal sealed class HistoricalMetricsResult
{
    [JsonPropertyName("text")]
    public string Text { get; set; } = string.Empty;

    [JsonPropertyName("keywordMetrics")]
    public RawKeywordMetrics KeywordMetrics { get; set; } = new();
}

internal sealed class RawKeywordMetrics
{
    [JsonPropertyName("avgMonthlySearches")]
    public string? AvgMonthlySearches { get; set; }

    [JsonPropertyName("competition")]
    public string Competition { get; set; } = string.Empty;

    [JsonPropertyName("competitionIndex")]
    public string? CompetitionIndex { get; set; }

    [JsonPropertyName("lowTopOfPageBidMicros")]
    public string? LowTopOfPageBidMicros { get; set; }

    [JsonPropertyName("highTopOfPageBidMicros")]
    public string? HighTopOfPageBidMicros { get; set; }

    [JsonPropertyName("monthlySearchVolumes")]
    public List<RawMonthlyVolume>? MonthlySearchVolumes { get; set; }
}

internal sealed class RawMonthlyVolume
{
    [JsonPropertyName("year")]
    public string? Year { get; set; }

    [JsonPropertyName("month")]
    public string Month { get; set; } = string.Empty;

    [JsonPropertyName("monthlySearches")]
    public string? MonthlySearches { get; set; }
}

internal sealed class GenerateForecastMetricsRequest
{
    [JsonPropertyName("campaign")]
    public CampaignForecast Campaign { get; set; } = new();

    [JsonPropertyName("forecastPeriod")]
    public ForecastPeriod ForecastPeriod { get; set; } = new();

    [JsonPropertyName("currencyCode")]
    public string? CurrencyCode { get; set; }
}

internal sealed class ForecastPeriod
{
    [JsonPropertyName("startDate")]
    public string StartDate { get; set; } = string.Empty;

    [JsonPropertyName("endDate")]
    public string EndDate { get; set; } = string.Empty;
}

internal sealed class CampaignForecast
{
    [JsonPropertyName("languageConstants")]
    public IReadOnlyList<string>? LanguageConstants { get; set; }

    [JsonPropertyName("geoTargetConstants")]
    public IReadOnlyList<string>? GeoTargetConstants { get; set; }

    [JsonPropertyName("biddingStrategy")]
    public BiddingStrategy BiddingStrategy { get; set; } = new();

    [JsonPropertyName("adGroups")]
    public IReadOnlyList<AdGroupForecast> AdGroups { get; set; } = [];
}

internal sealed class BiddingStrategy
{
    [JsonPropertyName("manualCpcBiddingStrategy")]
    public ManualCpcBiddingStrategy ManualCpcBiddingStrategy { get; set; } = new();
}

internal sealed class ManualCpcBiddingStrategy
{
    [JsonPropertyName("maxCpcBidMicros")]
    public string MaxCpcBidMicros { get; set; } = string.Empty;
}

internal sealed class AdGroupForecast
{
    [JsonPropertyName("keywords")]
    public IReadOnlyList<ForecastKeyword> Keywords { get; set; } = [];
}

internal sealed class ForecastKeyword
{
    [JsonPropertyName("text")]
    public string Text { get; set; } = string.Empty;

    [JsonPropertyName("matchType")]
    public string MatchType { get; set; } = "BROAD";
}

internal sealed class GenerateForecastMetricsResponse
{
    [JsonPropertyName("campaignForecastMetrics")]
    public ForecastMetricData? CampaignForecastMetrics { get; set; }
}

internal sealed class ForecastMetricData
{
    [JsonPropertyName("clicks")]
    public double? Clicks { get; set; }

    [JsonPropertyName("costMicros")]
    [JsonNumberHandling(JsonNumberHandling.AllowReadingFromString)]
    public long? CostMicros { get; set; }

    [JsonPropertyName("averageCpcMicros")]
    [JsonNumberHandling(JsonNumberHandling.AllowReadingFromString)]
    public long? AverageCpcMicros { get; set; }
}

internal sealed class TokenResponse
{
    [JsonPropertyName("access_token")]
    public string AccessToken { get; set; } = string.Empty;

    [JsonPropertyName("expires_in")]
    public int ExpiresIn { get; set; }
}

/// <summary>System.Text.Json source generation context for AOT-safe serialization.</summary>
[JsonSerializable(typeof(GenerateKeywordIdeasRequest))]
[JsonSerializable(typeof(GenerateKeywordIdeasResponse))]
[JsonSerializable(typeof(GenerateHistoricalMetricsRequest))]
[JsonSerializable(typeof(GenerateHistoricalMetricsResponse))]
[JsonSerializable(typeof(GenerateForecastMetricsRequest))]
[JsonSerializable(typeof(GenerateForecastMetricsResponse))]
[JsonSerializable(typeof(TokenResponse))]
[JsonSerializable(typeof(KeywordIdeasResponse))]
[JsonSerializable(typeof(HistoricalMetricsResponse))]
[JsonSerializable(typeof(ForecastResponse))]
[JsonSerializable(typeof(ErrorResult))]
[JsonSourceGenerationOptions(
    PropertyNamingPolicy = JsonKnownNamingPolicy.CamelCase,
    WriteIndented = false,
    DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull)]
internal partial class KwpJsonContext : JsonSerializerContext;
