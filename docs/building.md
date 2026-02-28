# Building from Source

## Go

**Requirements:** Go 1.21+

```bash
cd go
go build -o kwp-mcp-go .
```

The output binary is `kwp-mcp-go` (or `kwp-mcp-go.exe` on Windows if you add `.exe`).

---

## C# Native AOT

**Requirements:** .NET 10 SDK, platform-appropriate native build tools (e.g. Visual Studio Build Tools on Windows, `gcc` on Linux)

```bash
cd csharp
dotnet publish src/KeywordPlannerMcp/KeywordPlannerMcp.csproj -r <RID> -c Release --self-contained true
```

Replace `<RID>` with your target Runtime Identifier:

| Platform | RID |
|----------|-----|
| Linux x64 | `linux-x64` |
| Linux ARM64 | `linux-arm64` |
| macOS x64 | `osx-x64` |
| macOS ARM64 (Apple Silicon) | `osx-arm64` |
| Windows x64 | `win-x64` |

The published binary is in `csharp/src/KeywordPlannerMcp/bin/Release/net10.0/<RID>/publish/`.

---

## Running Tests

```bash
# Go
cd go && go test ./...

# C#
cd csharp && dotnet test
```
