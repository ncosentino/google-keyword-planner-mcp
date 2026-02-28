---
description: All three MCP tools exposed by the Google Keyword Planner MCP server -- generate keyword ideas, get historical metrics, and forecast keyword performance.
---

# MCP Tools

The Google Keyword Planner MCP server exposes three tools. All tools require valid credentials -- see [Configuration](../configuration.md).

## Tool Overview

| Tool | Purpose |
|------|---------|
| [`generate_keyword_ideas`](generate-keyword-ideas/) | Generate related keywords from seed keywords and/or a URL, with search volume and CPC data |
| [`get_historical_metrics`](get-historical-metrics/) | Get historical search volume, competition, and CPC for a specific list of keywords |
| [`get_keyword_forecast`](get-keyword-forecast/) | Project impressions, clicks, and cost for keywords at a given max CPC bid |

## Common Notes

**Search volume precision:** Without active ad spend on the account, Google returns search volumes as ranges rather than exact numbers. Any minimal ad spend unlocks precise monthly volumes.

**Rate limits:** The Keyword Planner API has quotas. For large batches, prefer `get_historical_metrics` with a list of known keywords over multiple `generate_keyword_ideas` calls.

**Language codes:** Language is specified as a resource name like `languageConstants/1000` (English). See the [Google Ads API reference](https://developers.google.com/google-ads/api/data/codes-formats#languages) for other language codes.
