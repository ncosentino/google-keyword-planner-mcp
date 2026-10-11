---
description: Compare the Go and C# Native AOT implementations of the Google Keyword Planner MCP server -- binary size, startup time, and when to choose each.
---

# Go vs C#

Both implementations expose keyword tools and share the transport contract,
but their existing argument formats are deliberately different. Both use
repaired Google Ads v25 contracts and per-call and default targeting. Integration
evidence is tracked in [#32](https://github.com/ncosentino/google-keyword-planner-mcp/issues/32).
Go accepts snake_case arguments and keyword arrays; C# preserves camelCase
arguments and comma-separated keyword strings.

| Behavior | Go | C# |
|---|---|---|
| Historical v25 `results` and string integers | Supported | Supported |
| Forecast joint total, optional metrics, inclusive future dates | Supported | Supported |
| Explicit unrestricted targeting overrides | Supported | Supported |
| ISO/name resolution and location discovery | Supported | Supported |
| Complete idea pagination before ranking/filtering/limiting | Supported | Supported |
| Optional monthly volumes and `totalFound` | Supported | Supported |
| Opt-in forecast scenarios, at most five workers, cancellation | Supported | Supported |
| Shared planning pace within a client | One request/second | One request/second |

Existing wire-format differences remain: Go may omit zero historical bid
estimates and empty monthly-volume arrays; C# includes those values. Forecast
optional metrics preserve absent versus zero in both. `competitionIndex: 0`
is included in both idea responses. Empty language strings mean omission;
use `all` to remove a configured language. Explicit worldwide geography is
an empty array in Go and an empty string in C#.

Quota pacing is local to one configured client/process, not a distributed
customer-level limiter. Multiple instances sharing a Google Ads customer must
coordinate their aggregate traffic.

CI consumes shared synthetic v25 request/response fixtures in both suites,
checks Go data races, and publishes C# Native AOT. These checks do not substitute
for an authenticated live API probe.

## Comparison

| | Go | C# Native AOT |
|---|---|---|
| Binary size | ~12 MB | ~20 MB |
| Startup time | < 5 ms | ~20 ms |
| Memory usage | Lower | Slightly higher |
| Platforms | All | All |
| Runtime required | None | None |
| Transports | stdio, HTTP | stdio, HTTP |
| HTTP mode | Stateless | Stateless |
| Default listener | `127.0.0.1:8080` | `127.0.0.1:8080` |
| HTTP endpoints | `/mcp`, `/health` | `/mcp`, `/health` |

Both are self-contained native binaries. Both implement the same shared HTTP
contract, including Host and Origin validation. The only cosmetic difference
is the disallowed-Host status code (`403` for Go and `400` for C#).

## When to Choose Go

- You prefer smaller binary sizes.
- You're deploying on resource-constrained machines where startup time matters.
- You want to build from source using only the Go toolchain.

## When to Choose C#

- You're already in a .NET ecosystem and prefer to build from source with `dotnet publish`.
- Your organization has tooling that works better with .NET binaries.

## Which Binary Is Right for Most Users?

For AI assistant integration (GitHub Copilot, Claude, Cursor), either works fine. The server starts once and stays running -- the ~15 ms startup difference is imperceptible. Pick whichever binary matches the platform you're on (see the [Getting Started](getting-started.md) download table).
