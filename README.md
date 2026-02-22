# google-keyword-planner-mcp

[![CI](https://github.com/ncosentino/google-keyword-planner-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/ncosentino/google-keyword-planner-mcp/actions/workflows/ci.yml)
[![Release](https://github.com/ncosentino/google-keyword-planner-mcp/actions/workflows/release.yml/badge.svg)](https://github.com/ncosentino/google-keyword-planner-mcp/releases/latest)

A zero-dependency native MCP server for the [Google Ads Keyword Planner API](https://developers.google.com/google-ads/api/docs/keyword-planning/overview).

Available as a statically-linked Go binary **or** a C# Native AOT binary -- no runtime, no Docker, no Node.js.

---

## Available Tools

| Tool | Description |
|------|-------------|
| `generate_keyword_ideas` | Generate related keywords from seed keywords and/or a URL with search volume and CPC data |
| `get_historical_metrics` | Get historical search volume, competition, and CPC for a list of keywords |
| `get_keyword_forecast` | Get projected impressions, clicks, and cost for keywords at a given max CPC bid |

---

## Quick Start

### 1. Prerequisites

You need:

1. A **Google Cloud project** with the Google Ads API enabled
2. A **Google Ads developer token** -- apply at [Google Ads API Center](https://ads.google.com/intl/en_us/home/tools/keyword-planner/)
3. An **OAuth2 client ID and secret** from your GCP project (Desktop app type)
4. An **OAuth2 refresh token** obtained via a one-time authorization flow (see below)
5. A **Google Ads customer ID** (your account ID, format: `123-456-7890`)

### 2. Obtain a Refresh Token

Run a one-time OAuth2 flow to get a refresh token:

```bash
# Exchange your authorization code for tokens
# Step 1: Visit this URL in your browser (replace CLIENT_ID):
https://accounts.google.com/o/oauth2/v2/auth?client_id=CLIENT_ID&redirect_uri=urn:ietf:wg:oauth:2.0:oob&response_type=code&scope=https://www.googleapis.com/auth/adwords&access_type=offline&prompt=consent

# Step 2: Exchange the code for tokens (replace CLIENT_ID, CLIENT_SECRET, CODE):
curl -X POST https://oauth2.googleapis.com/token \
  -d "client_id=CLIENT_ID" \
  -d "client_secret=CLIENT_SECRET" \
  -d "code=CODE" \
  -d "grant_type=authorization_code" \
  -d "redirect_uri=urn:ietf:wg:oauth:2.0:oob"
```

The `refresh_token` field in the response is what you need.

### 3. Download the Binary

Download the latest binary for your platform from the [Releases page](https://github.com/ncosentino/google-keyword-planner-mcp/releases/latest).

| Platform | Go binary | C# AOT binary |
|----------|-----------|---------------|
| Linux x64 | `kwp-mcp-go-linux-amd64` | `kwp-mcp-csharp-linux-x64` |
| Linux ARM64 | `kwp-mcp-go-linux-arm64` | `kwp-mcp-csharp-linux-arm64` |
| macOS x64 | `kwp-mcp-go-darwin-amd64` | `kwp-mcp-csharp-osx-x64` |
| macOS ARM64 | `kwp-mcp-go-darwin-arm64` | `kwp-mcp-csharp-osx-arm64` |
| Windows x64 | `kwp-mcp-go-windows-amd64.exe` | `kwp-mcp-csharp-win-x64.exe` |
| Windows ARM64 | `kwp-mcp-go-windows-arm64.exe` | `kwp-mcp-csharp-win-arm64.exe` |

### 4. Configure Your MCP Client

#### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "keyword-planner": {
      "command": "/path/to/kwp-mcp-go-linux-amd64",
      "env": {
        "GOOGLE_ADS_DEVELOPER_TOKEN": "your-developer-token",
        "GOOGLE_ADS_CLIENT_ID": "your-client-id.apps.googleusercontent.com",
        "GOOGLE_ADS_CLIENT_SECRET": "your-client-secret",
        "GOOGLE_ADS_REFRESH_TOKEN": "your-refresh-token",
        "GOOGLE_ADS_CUSTOMER_ID": "123-456-7890"
      }
    }
  }
}
```

---

## Configuration Reference

Credentials are resolved in this priority order: **CLI flag > environment variable > `.env` file**.

| Credential | CLI flag | Environment variable | Description |
|------------|----------|---------------------|-------------|
| Developer token | `--developer-token` | `GOOGLE_ADS_DEVELOPER_TOKEN` | From Google Ads API Center |
| OAuth2 client ID | `--client-id` | `GOOGLE_ADS_CLIENT_ID` | From GCP OAuth2 credentials |
| OAuth2 client secret | `--client-secret` | `GOOGLE_ADS_CLIENT_SECRET` | From GCP OAuth2 credentials |
| Refresh token | `--refresh-token` | `GOOGLE_ADS_REFRESH_TOKEN` | From one-time OAuth2 flow |
| Customer ID | `--customer-id` | `GOOGLE_ADS_CUSTOMER_ID` | Google Ads account ID |

### `.env` File

Place a `.env` file in the same directory as the binary:

```env
GOOGLE_ADS_DEVELOPER_TOKEN=your-developer-token
GOOGLE_ADS_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_ADS_CLIENT_SECRET=your-client-secret
GOOGLE_ADS_REFRESH_TOKEN=your-refresh-token
GOOGLE_ADS_CUSTOMER_ID=123-456-7890
```

---

## Tool Reference

### `generate_keyword_ideas`

Generates related keyword ideas from seed keywords and/or a URL.

**Parameters:**
- `seedKeywords` (optional) -- comma-separated seed keywords (e.g. `"C# tutorial, dotnet performance"`)
- `url` (optional) -- a URL to generate keyword ideas from (e.g. `"https://devleader.ca"`)
- `language` (optional) -- language resource name (e.g. `"languageConstants/1000"` for English)

At least one of `seedKeywords` or `url` must be provided.

**Returns:** List of keyword ideas with `avgMonthlySearches`, `competition` (LOW/MEDIUM/HIGH), `lowTopOfPageBidMicros`, `highTopOfPageBidMicros`.

**Note:** Without active ad spend, search volumes are shown as ranges. Any minimal ad spend unlocks precise monthly volumes.

---

### `get_historical_metrics`

Gets historical search volume and competition data for a specific list of keywords.

**Parameters:**
- `keywords` (required) -- comma-separated list of keywords (e.g. `"dependency injection, SOLID principles"`)

**Returns:** Per-keyword metrics including `avgMonthlySearches`, `competition`, `competitionIndex`, bid estimates, and `monthlySearchVolumes` (12-month breakdown).

---

### `get_keyword_forecast`

Projects impressions, clicks, and cost for keywords at a specified maximum CPC bid.

**Parameters:**
- `keywords` (required) -- comma-separated list of keywords
- `maxCpcMicros` (optional, default `1000000`) -- maximum CPC bid in micros (1,000,000 = $1.00)
- `forecastDays` (optional, default `30`) -- number of days to forecast

**Returns:** Per-keyword projected `impressions`, `clicks`, `costMicros`, and `ctr`.

---

## Go vs C# -- Which Binary to Use?

Both binaries implement identical behavior. Choose based on preference:

| | Go | C# AOT |
|---|---|---|
| Binary size | ~12 MB | ~20 MB |
| Startup time | < 5ms | ~20ms |
| Memory usage | Lower | Slightly higher |
| Platform | All | All |

For most MCP use cases, either works fine. The Go binary starts slightly faster; the C# binary may be preferred if you're already in a .NET ecosystem.

---

## Building from Source

### Go

```bash
cd go
go build -o kwp-mcp-go .
```

### C#

```bash
cd csharp
dotnet publish src/KeywordPlannerMcp/KeywordPlannerMcp.csproj -r linux-x64 -c Release --self-contained true
```

Replace `linux-x64` with your target RID (`win-x64`, `osx-arm64`, etc.).

---

## Related Projects

- [google-psi-mcp](https://github.com/ncosentino/google-psi-mcp) -- Google PageSpeed Insights MCP server
- [google-search-console-mcp](https://github.com/ncosentino/google-search-console-mcp) -- Google Search Console MCP server

---

## About

Built to give AI assistants direct access to Google's keyword database for content strategy, SEO gap analysis, and search volume research.

Part of the [Nexus Labs](https://github.com/ncosentino) native MCP server collection -- zero dependencies, single binary downloads.

---

## Contributing

Issues and pull requests welcome. Please run `go test ./...` and `dotnet test` before submitting.

---

## License

MIT License -- see [LICENSE](LICENSE).
