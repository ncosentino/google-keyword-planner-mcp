---
applyTo: "go/**/*_test.go"
---

Use channel barriers and explicit cancellation to coordinate concurrent HTTP
tests. Do not use sleeps to assume request ordering or overlap. Verify both
the concurrency ceiling and caller cancellation; cancelled work must not
return partial forecasts as successful results.

Model Google Ads fixtures on the pinned API contract. Historical results use
`results`; forecasts use `campaignForecastMetrics`. Exercise nonempty payloads
and distinguish missing optional metrics from actual numeric zero values.
