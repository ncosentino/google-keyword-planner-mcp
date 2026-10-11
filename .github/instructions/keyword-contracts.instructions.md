---
applyTo: "go/internal/keywordplanner/**,csharp/src/KeywordPlannerMcp/KeywordPlanner/**,csharp/tests/KeywordPlannerMcp.Tests/**"
---

Treat C# and Go as implementations of the same functional API contract while
preserving their documented legacy argument formats. Temporary divergence
requires evidence-backed tracking issues and companion implementation PRs.

Use the shared synthetic v25 response fixtures in `testdata/google-ads/v25`
from both test suites. Update them against authoritative Google API definitions,
not the current implementation. Preserve optional metric presence and assert
nonempty results so an incorrect response key cannot pass unnoticed.

Coordinate concurrency tests with barriers, cancellation and fake time, not
sleep-based ordering assumptions. Verify cancellation never returns partial
forecasts as success. Planning concurrency does not replace customer-level
quota pacing.
