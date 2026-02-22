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
    string Text,
    double Impressions,
    double Clicks,
    double CostMicros,
    double Ctr);

/// <summary>The result of a keyword forecast request.</summary>
internal sealed record ForecastResponse(
    IReadOnlyList<KeywordForecastMetrics> Keywords,
    int ForecastDays,
    long MaxCpcMicros);

/// <summary>An error result.</summary>
internal sealed record ErrorResult(string Error);

// --- Raw Google Ads API request/response models ---

internal sealed class GenerateKeywordIdeasRequest
{
    [JsonPropertyName("language")]
    public string? Language { get; set; }

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
}

internal sealed class GenerateHistoricalMetricsResponse
{
    [JsonPropertyName("metrics")]
    public List<HistoricalMetricsResult>? Metrics { get; set; }
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
    public int CompetitionIndex { get; set; }

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
    public int Year { get; set; }

    [JsonPropertyName("month")]
    public string Month { get; set; } = string.Empty;

    [JsonPropertyName("monthlySearches")]
    public string? MonthlySearches { get; set; }
}

internal sealed class GenerateForecastMetricsRequest
{
    [JsonPropertyName("campaignForecastSpec")]
    public CampaignForecastSpec CampaignForecastSpec { get; set; } = new();
}

internal sealed class CampaignForecastSpec
{
    [JsonPropertyName("biddingStrategy")]
    public BiddingStrategy BiddingStrategy { get; set; } = new();

    [JsonPropertyName("startDate")]
    public string StartDate { get; set; } = string.Empty;

    [JsonPropertyName("endDate")]
    public string EndDate { get; set; } = string.Empty;

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
    [JsonPropertyName("biddableKeywords")]
    public IReadOnlyList<AdGroupForecastKeyword> BiddableKeywords { get; set; } = [];
}

internal sealed class AdGroupForecastKeyword
{
    [JsonPropertyName("keyword")]
    public ForecastKeyword Keyword { get; set; } = new();
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
    [JsonPropertyName("adGroupForecastMetrics")]
    public List<AdGroupForecastMetrics>? AdGroupForecastMetrics { get; set; }
}

internal sealed class AdGroupForecastMetrics
{
    [JsonPropertyName("keywordForecastMetrics")]
    public List<KeywordForecastMetric>? KeywordForecastMetrics { get; set; }
}

internal sealed class KeywordForecastMetric
{
    [JsonPropertyName("keyword")]
    public ForecastKeyword Keyword { get; set; } = new();

    [JsonPropertyName("metrics")]
    public ForecastMetricData Metrics { get; set; } = new();
}

internal sealed class ForecastMetricData
{
    [JsonPropertyName("impressions")]
    public double Impressions { get; set; }

    [JsonPropertyName("clicks")]
    public double Clicks { get; set; }

    [JsonPropertyName("costMicros")]
    public double CostMicros { get; set; }

    [JsonPropertyName("ctr")]
    public double Ctr { get; set; }
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
