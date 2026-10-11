---
description: Forecast clicks and cost for a combined keyword campaign using Google Ads v25.
---

# get_keyword_forecast

Both implementations forecast a combined ad group using Google Ads v25.
The following uses Go's snake_case names and array inputs. C# uses camelCase
names and comma-separated `keywords` and `geoTargetConstants` strings.
For C#, an empty geography string explicitly requests worldwide targeting.

## Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `keywords` | string array | Required | Keywords to forecast |
| `max_cpc_micros` | integer | `1000000` | Manual CPC bid in micros of the requested or account currency |
| `forecast_days` | integer | `30` | Inclusive forecast length, at most 364 days |
| `match_type` | string | `BROAD` | `BROAD`, `PHRASE`, or `EXACT` |
| `language` | string | Server default | Language resource name; `all` overrides the default |
| `geo_target_constants` | string array | Server default | Resource names; `[]` explicitly requests worldwide targeting |
| `currency_code` | string | Account currency | Currency code for the returned costs |
| `include_keyword_breakdown` | boolean | `false` | Additional independent keyword scenarios |

## Response

`total` contains the campaign-level forecast. `keywords` is empty unless the
isolated breakdown is requested. Optional numeric metrics are omitted when
unavailable, rather than represented as zero.

```json
{
  "keywords": [],
  "total": {"clicks": 350, "costMicros": 280000000, "averageCpcMicros": 800000},
  "forecastDays": 30,
  "maxCpcMicros": 1000000,
  "startDate": "2026-10-13",
  "endDate": "2026-11-11",
  "matchType": "BROAD"
}
```

The forecast starts two UTC calendar days ahead so it is future-dated in any
account timezone. Both endpoints are inclusive. Impressions and CTR were
removed from the v25 forecast schema. Conversion predictions are not available
for the manual-CPC bidding strategy used by this tool.

Independent single-keyword scenarios are not additive contributions to the
combined campaign: keywords compete for the same traffic. Enabling the
breakdown adds one API call per keyword. Planning calls share a per-client
one-request-per-second gate for the configured customer; queued calls honor
cancellation.

Costs divided by 1,000,000 are currency units, not necessarily US dollars.
Forecasts are estimates, not guaranteed outcomes.

See Google's [forecast contract](https://developers.google.com/google-ads/api/docs/keyword-planning/generate-forecast-metrics)
and [planning quotas](https://developers.google.com/google-ads/api/docs/best-practices/quotas).
