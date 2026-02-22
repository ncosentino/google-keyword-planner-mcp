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
        [Description("Language resource name (e.g. 'languageConstants/1000' for English). Omit to use all languages.")] string? language = null,
        CancellationToken cancellationToken = default)
    {
        var seeds = string.IsNullOrWhiteSpace(seedKeywords)
            ? null
            : seedKeywords.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);

        try
        {
            var result = await client.GenerateKeywordIdeasAsync(seeds, url, language, cancellationToken)
                .ConfigureAwait(false);
            return JsonSerializer.Serialize(result, KwpJsonContext.Default.KeywordIdeasResponse);
        }
        catch (Exception ex)
        {
            var err = new ErrorResult($"Error generating keyword ideas: {ex.Message}");
            return JsonSerializer.Serialize(err, KwpJsonContext.Default.ErrorResult);
        }
    }

    [McpServerTool(Name = "get_historical_metrics")]
    [Description("Get historical search volume and competition metrics for a list of keywords using Google Ads Keyword Planner.")]
    internal async Task<string> GetHistoricalMetricsAsync(
        [Description("Comma-separated list of keywords to look up (e.g. 'dependency injection, SOLID principles').")] string keywords,
        CancellationToken cancellationToken = default)
    {
        var keywordList = keywords.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);

        try
        {
            var result = await client.GetHistoricalMetricsAsync(keywordList, cancellationToken)
                .ConfigureAwait(false);
            return JsonSerializer.Serialize(result, KwpJsonContext.Default.HistoricalMetricsResponse);
        }
        catch (Exception ex)
        {
            var err = new ErrorResult($"Error getting historical metrics: {ex.Message}");
            return JsonSerializer.Serialize(err, KwpJsonContext.Default.ErrorResult);
        }
    }

    [McpServerTool(Name = "get_keyword_forecast")]
    [Description("Get projected impressions, clicks, and cost for a set of keywords at a given max CPC bid using Google Ads Keyword Planner. max_cpc_micros is in micros (1,000,000 = $1.00). forecast_days defaults to 30.")]
    internal async Task<string> GetKeywordForecastAsync(
        [Description("Comma-separated list of keywords to forecast.")] string keywords,
        [Description("Maximum CPC bid in micros (1,000,000 = $1.00). Defaults to 1,000,000.")] long maxCpcMicros = 1_000_000,
        [Description("Number of days to forecast. Defaults to 30.")] int forecastDays = 30,
        CancellationToken cancellationToken = default)
    {
        var keywordList = keywords.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);

        try
        {
            var result = await client.GetKeywordForecastAsync(keywordList, maxCpcMicros, forecastDays, cancellationToken)
                .ConfigureAwait(false);
            return JsonSerializer.Serialize(result, KwpJsonContext.Default.ForecastResponse);
        }
        catch (Exception ex)
        {
            var err = new ErrorResult($"Error getting keyword forecast: {ex.Message}");
            return JsonSerializer.Serialize(err, KwpJsonContext.Default.ErrorResult);
        }
    }
}
