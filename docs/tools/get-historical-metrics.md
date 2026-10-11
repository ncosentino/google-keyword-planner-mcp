---
description: get_historical_metrics MCP tool -- retrieve historical search volume, competition scores, and CPC estimates for a list of keywords, including 12-month breakdowns.
---

# get_historical_metrics

Get historical search volume and competition data for a specific list of keywords. Unlike `generate_keyword_ideas`, this tool takes an exact keyword list and returns data for each one -- no new ideas are generated.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `keywords` | string | Yes | Comma-separated list of exact keywords (e.g. `"dependency injection, SOLID principles"`) |
| `language` | string | No | ISO code, name, ID, or resource name; `all` removes a default |
| `geoTargetConstants` | string | No | Comma-separated geo values; empty string requests worldwide |
| `keywordPlanNetwork` | string | No | `GOOGLE_SEARCH` or `GOOGLE_SEARCH_AND_PARTNERS` |

The table uses C# names and strings. Go uses keyword arrays,
`geo_target_constants` arrays, and `keyword_plan_network`. Omitted targeting
uses configured defaults in both implementations.

## Response

Returns a list of objects, one per keyword:

```json
{
  "text": "dependency injection",
  "avgMonthlySearches": 22200,
  "competition": "LOW",
  "competitionIndex": 18,
  "lowTopOfPageBidMicros": 500000,
  "highTopOfPageBidMicros": 2100000,
  "monthlySearchVolumes": [
    { "year": 2024, "month": 1, "monthlySearches": 21000 },
    { "year": 2024, "month": 2, "monthlySearches": 23000 }
  ]
}
```

| Field | Description |
|-------|-------------|
| `text` | The keyword text |
| `avgMonthlySearches` | 12-month average monthly search volume |
| `competition` | `LOW`, `MEDIUM`, or `HIGH` |
| `competitionIndex` | Competition score 0-100 |
| `lowTopOfPageBidMicros` | Low-end CPC bid estimate in micros |
| `highTopOfPageBidMicros` | High-end CPC bid estimate in micros |
| `monthlySearchVolumes` | Month-by-month breakdown for the past 12 months |

## Example Prompts

- _"What are the historical search volumes for: dependency injection, SOLID principles, clean architecture?"_
- _"How competitive are these keywords: C# async await, dotnet performance?"_
- _"Get search metrics for a list of keywords I'm targeting in my blog."_
- _"Check the 12-month trend for 'software architecture' and 'design patterns'."_

## Notes

- The response wraps entries in `keywords` and includes `count`. Monthly values
  are numeric month indexes, even though the Google response uses enum names.
- Up to a few thousand keywords can be passed in a single call.
- `monthlySearchVolumes` is particularly useful for spotting seasonal trends.
