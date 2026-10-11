---
description: Discover country, region, and city geo target resources for keyword planning in Go and C#.
---

# search_geo_targets

Find locations by name and use their resource names in the keyword tools.

| Parameter | Go | C# |
|---|---|---|
| `names` (required) | Array, such as `["Vienna", "Salzburg"]` | String, such as `"Vienna, Salzburg"` |
| Optional country filter | `country_code: "AT"` | `countryCode: "AT"` |
| Optional locale | `locale: "en"` | `locale: "en"` |

Locale defaults to `en`. Names must be nonblank. Country filters are ISO codes.
The response contains `geoTargets` and `count`; each target exposes
`resourceName`, `name`, `canonicalName`, `countryCode`, `targetType`, and `reach`.

Use a returned resource name directly for regions and cities. Country ISO
codes can also be resolved by the keyword tools. Suggestions use Google's
global v25 `geoTargetConstants:suggest` endpoint, not a customer planning
endpoint.
