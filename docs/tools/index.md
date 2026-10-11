---
description: Four MCP tools for keyword ideas, historical metrics, forecasts, and location discovery in both implementations.
---

# MCP Tools

Both implementations expose four tools. All tools require valid credentials -- see [Configuration](../configuration.md).

## Tool Overview

| Tool | Purpose |
|------|---------|
| [`generate_keyword_ideas`](generate-keyword-ideas/) | Generate related keywords from seed keywords and/or a URL, with search volume and CPC data |
| [`get_historical_metrics`](get-historical-metrics/) | Get historical search volume, competition, and CPC for a specific list of keywords |
| [`get_keyword_forecast`](get-keyword-forecast/) | Project available clicks, cost, and average CPC at a given max CPC bid |
| [`search_geo_targets`](search-geo-targets/) | Discover country, region, and city target resource names |

## Common Notes

**Search volume precision:** Without active ad spend on the account, Google returns search volumes as ranges rather than exact numbers. Any minimal ad spend unlocks precise monthly volumes.

**Rate limits:** Planning calls share a conservative one-request-per-second
pace within a client/process. Multiple instances using the same customer must
coordinate aggregate traffic. Optional forecast breakdowns require extra calls.

**Language codes:** Both accept resource names, numeric IDs, ISO codes, and
English names. Use `all` to remove a configured restriction.
