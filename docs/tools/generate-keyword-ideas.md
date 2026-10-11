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
| `geoTargetConstants` | string | No | Comma-separated IDs, ISO country codes, names, or resource names; empty string means worldwide |
| `keywordPlanNetwork` | string | No | `GOOGLE_SEARCH` or `GOOGLE_SEARCH_AND_PARTNERS` |
| `limit` | integer | No | Default 50; zero also means 50; -1 returns all |
| `minMonthlySearches` | integer | No | Nonnegative minimum average monthly volume |
| `includeMonthlyVolumes` | boolean | No | Include available month-by-month volumes; default false |

\* At least one of `seedKeywords` or `url` must be provided.

This table uses C# names. Go uses `seed_keywords`, `geo_target_constants`,
`keyword_plan_network`, `min_monthly_searches`, and `include_monthly_volumes`;
keyword seeds and geography are arrays rather than comma-separated strings.
Language accepts ISO codes, English names, bare IDs, or resource names in both.
Omitted targeting uses configured defaults; `language: "all"` removes its
restriction. Explicit worldwide geography is `[]` in Go.

## Response

Returns a list of keyword ideas. Each entry includes:

```json
{
  "text": "dependency injection c#",
  "avgMonthlySearches": 1900,
  "competition": "MEDIUM",
  "competitionIndex": 52,
  "lowTopOfPageBidMicros": 800000,
  "highTopOfPageBidMicros": 3200000
}
```

| Field | Description |
|-------|-------------|
| `text` | The keyword text |
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
- All API pages are fetched before stable volume sorting, filtering, and limiting.
- The response wraps entries in `ideas`, includes returned `count`, and uses
  `totalFound` for the fetched count before filtering/truncation.
- Without configured defaults, omitted language means unrestricted. Use `all`
  to override a configured default.
- Negative minimum volumes or limits below -1 are errors.
