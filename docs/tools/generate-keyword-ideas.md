---
description: generate_keyword_ideas MCP tool -- generate related keywords from seed keywords or a URL with search volume, competition level, and CPC estimates.
---

# generate_keyword_ideas

Generate related keyword ideas from seed keywords and/or a URL. Returns keywords with average monthly search volume, competition level, and CPC estimates.

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `seedKeywords` | string | No* | Comma-separated seed keywords (e.g. `"C# tutorial, dotnet performance"`) |
| `url` | string | No* | A URL to generate ideas from (e.g. `"https://devleader.ca"`) |
| `language` | string | No | Language resource name (e.g. `"languageConstants/1000"` for English) |

\* At least one of `seedKeywords` or `url` must be provided.

## Response

Returns a list of keyword ideas. Each entry includes:

```json
{
  "keyword": "dependency injection c#",
  "avgMonthlySearches": 1900,
  "competition": "MEDIUM",
  "competitionIndex": 52,
  "lowTopOfPageBidMicros": 800000,
  "highTopOfPageBidMicros": 3200000
}
```

| Field | Description |
|-------|-------------|
| `keyword` | The keyword text |
| `avgMonthlySearches` | Average monthly search volume (may be a range without ad spend) |
| `competition` | `LOW`, `MEDIUM`, or `HIGH` |
| `competitionIndex` | Competition score 0-100 |
| `lowTopOfPageBidMicros` | Low end of top-of-page CPC bid estimate in micros ($1 = 1,000,000) |
| `highTopOfPageBidMicros` | High end of top-of-page CPC bid estimate in micros |

## Example Prompts

- _"What keywords should I target for a blog post about dependency injection in C#?"_
- _"Find low-competition keywords related to software architecture."_
- _"Generate keyword ideas for this URL: https://www.devleader.ca/2024/01/01/my-post/"_
- _"What keywords with high search volume and low competition exist around .NET performance?"_

## Notes

- Without active ad spend on the associated account, `avgMonthlySearches` is returned as a range (e.g. 1000-10000) rather than an exact number. Minimal ad spend unlocks exact values.
- Omit `language` to get results across all languages.
- The API may return up to several hundred keyword ideas per call.
