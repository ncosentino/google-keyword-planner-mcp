# Go vs C#

Both implementations expose identical MCP tools and accept the same credentials. Choose based on your environment and preferences.

## Comparison

| | Go | C# Native AOT |
|---|---|---|
| Binary size | ~12 MB | ~20 MB |
| Startup time | < 5 ms | ~20 ms |
| Memory usage | Lower | Slightly higher |
| Platforms | All | All |
| Runtime required | None | None |

Both are self-contained native binaries -- no Go toolchain, no .NET runtime, no Node.js, no Python required.

## When to Choose Go

- You prefer smaller binary sizes.
- You're deploying on resource-constrained machines where startup time matters.
- You want to build from source using only the Go toolchain.

## When to Choose C#

- You're already in a .NET ecosystem and prefer to build from source with `dotnet publish`.
- Your organization has tooling that works better with .NET binaries.

## Which Binary Is Right for Most Users?

For AI assistant integration (GitHub Copilot, Claude, Cursor), either works fine. The server starts once and stays running -- the ~15 ms startup difference is imperceptible. Pick whichever binary matches the platform you're on (see the [Getting Started](getting-started.md) download table).
