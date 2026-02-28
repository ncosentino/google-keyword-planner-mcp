# get_historical_metrics

Get historical search volume and competition data for a specific list of keywords. Unlike `generate_keyword_ideas`, this tool takes an exact keyword list and returns data for each one -- no new ideas are generated.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `keywords` | string | Yes | Comma-separated list of exact keywords (e.g. `"dependency injection, SOLID principles"`) |

## Response

Returns a list of objects, one per keyword:

```json
{
  "keyword": "dependency injection",
  "avgMonthlySearches": 22200,
  "competition": "LOW",
  "competitionIndex": 18,
  "lowTopOfPageBidMicros": 500000,
  "highTopOfPageBidMicros": 2100000,
  "monthlySearchVolumes": [
    { "year": 2024, "month": "JANUARY", "monthlySearches": 21000 },
    { "year": 2024, "month": "FEBRUARY", "monthlySearches": 23000 }
  ]
}
```

| Field | Description |
|-------|-------------|
| `keyword` | The keyword text |
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

- Up to a few thousand keywords can be passed in a single call.
- `monthlySearchVolumes` is particularly useful for spotting seasonal trends.
