# Google Ads Keyword Planner MCP Server -- Keyword Research for AI Assistants

[![Latest Release](https://img.shields.io/github/v/release/ncosentino/google-keyword-planner-mcp?style=flat-square)](https://github.com/ncosentino/google-keyword-planner-mcp/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)](go/go.mod)
[![.NET Version](https://img.shields.io/badge/.NET-10-512BD4?style=flat-square&logo=dotnet)](csharp/Directory.Build.props)
[![CI](https://img.shields.io/github/actions/workflow/status/ncosentino/google-keyword-planner-mcp/ci.yml?label=CI&style=flat-square)](https://github.com/ncosentino/google-keyword-planner-mcp/actions/workflows/ci.yml)

> **Zero-dependency MCP server for Google Ads Keyword Planner.**
> Pre-built native binaries for Linux, macOS, and Windows. No Node.js. No Python. No .NET runtime. No Go toolchain. Download one binary and configure your AI tool.

Expose Google Ads Keyword Planner data directly to AI assistants like Claude, GitHub Copilot, and Cursor via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io). Ask your AI to find keyword opportunities, analyze search volume trends, and forecast ad performance -- all grounded in real Google keyword data.

---

## Why This Exists

AI assistants are powerful at SEO and content strategy -- but they need real keyword data. This MCP server bridges your AI tool to the Google Ads Keyword Planner API, giving it:

- **Keyword ideas** -- related keywords with average monthly searches, competition level, and CPC estimates from a seed keyword or URL
- **Historical metrics** -- search volume trends, competition scores, and bid ranges for any list of keywords
- **Forecasts** -- projected impressions, clicks, and cost for a set of keywords at a given max CPC bid

With this MCP server configured, you can ask your AI: _"What keywords should I target for a blog post about dependency injection in C#? What's the search volume and competition look like?"_ and get a real data-backed answer.

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

1. A **Google Ads manager account (MCC)** -- developer tokens are only issued to manager accounts, not regular accounts. Create one free at [ads.google.com/home/tools/manager-accounts](https://ads.google.com/home/tools/manager-accounts/) if you don't have one.

2. A **Google Ads developer token with Basic or Standard access** -- in your manager account, go to `https://ads.google.com/aw/apicenter` and copy your developer token.

   > **⚠️ New tokens start in test mode.** A brand-new developer token can only call the API against [Google Ads test accounts](https://developers.google.com/google-ads/api/docs/best-practices/test-accounts). Calls to any real account return `DEVELOPER_TOKEN_NOT_APPROVED` until you apply for Basic access. To apply: in the API Center (`https://ads.google.com/aw/apicenter`), click **Apply for Basic Access** and fill in the form. Google reviews requests within a few days. You do not need Standard access -- Basic access is sufficient for the Keyword Planner API.

3. A **Google Ads account with billing configured** -- the Keyword Planner API requires an account with an active payment method. You do not need to run any ads or spend money; you just need a payment method on file. This can be your manager account itself (if it has billing) or a separate sub-account.

   > **Note on `GOOGLE_ADS_LOGIN_CUSTOMER_ID`:** If the account you use for `GOOGLE_ADS_CUSTOMER_ID` is a sub-account managed through your manager account, set `GOOGLE_ADS_LOGIN_CUSTOMER_ID` to your manager account ID. If you are calling the account directly (i.e., the OAuth user has direct access to it without going through a manager), you can omit `GOOGLE_ADS_LOGIN_CUSTOMER_ID`.

4. A **Google Cloud project** with the [Google Ads API](https://console.cloud.google.com/apis/library/googleads.googleapis.com) enabled.
5. An **OAuth2 client ID and secret** from your GCP project (Desktop app type -- see GCP setup below).
6. An **OAuth2 refresh token** obtained via a one-time authorization flow (see below). The OAuth2 flow must be completed as the Google account that has access to the Google Ads account you intend to use.
7. A **Google Ads customer ID** -- the ID of the account with billing (found in the top-right corner of the Google Ads UI). Dashes are optional: `123-456-7890` and `1234567890` both work.

### 2. GCP Setup

1. Go to [GCP Console → APIs & Services → Credentials](https://console.cloud.google.com/apis/credentials)
2. Enable the **Google Ads API**: `https://console.cloud.google.com/apis/library/googleads.googleapis.com`
3. Configure the OAuth consent screen (`https://console.cloud.google.com/apis/credentials/consent`):
   - Select **External** → fill in app name and your email → **Save**
   - Add your own Google account as a **Test user**
   - Leave the app in **Testing** mode
4. Create an OAuth2 client: **+ Create Credentials → OAuth client ID → Desktop app** → Create
5. Copy the **Client ID** and **Client Secret**

### 3. Obtain a Refresh Token

Run this one-time flow to get a refresh token. It starts a local HTTP listener to automatically capture the authorization code:

**PowerShell (Windows):**

```powershell
$clientId = "YOUR_CLIENT_ID"
$clientSecret = "YOUR_CLIENT_SECRET"
$redirectUri = "http://localhost:9876"
$authUrl = "https://accounts.google.com/o/oauth2/v2/auth?client_id=$clientId&redirect_uri=$([Uri]::EscapeDataString($redirectUri))&response_type=code&scope=$([Uri]::EscapeDataString('https://www.googleapis.com/auth/adwords'))&access_type=offline&prompt=consent"

$listener = [System.Net.HttpListener]::new()
$listener.Prefixes.Add("$redirectUri/")
$listener.Start()

Start-Process $authUrl

$context = $listener.GetContext()
$rawUrl = $context.Request.RawUrl
$responseText = "<html><body><h2>Auth complete! You can close this tab.</h2></body></html>"
$buffer = [System.Text.Encoding]::UTF8.GetBytes($responseText)
$context.Response.ContentLength64 = $buffer.Length
$context.Response.OutputStream.Write($buffer, 0, $buffer.Length)
$context.Response.Close()
$listener.Stop()

$code = ($rawUrl -split "[?&]" | Where-Object { $_ -like "code=*" }) -replace "^code=", ""

$body = "client_id=$clientId&client_secret=$clientSecret&code=$([Uri]::EscapeDataString($code))&grant_type=authorization_code&redirect_uri=$([Uri]::EscapeDataString($redirectUri))"
$result = Invoke-RestMethod -Method Post -Uri "https://oauth2.googleapis.com/token" -Body $body -ContentType "application/x-www-form-urlencoded"
Write-Host "Refresh token: $($result.refresh_token)"
```

**bash (Linux/macOS):**

```bash
CLIENT_ID="YOUR_CLIENT_ID"
CLIENT_SECRET="YOUR_CLIENT_SECRET"
REDIRECT_URI="http://localhost:9876"

# Open auth URL in browser
open "https://accounts.google.com/o/oauth2/v2/auth?client_id=${CLIENT_ID}&redirect_uri=${REDIRECT_URI}&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fadwords&access_type=offline&prompt=consent"

# Start a temporary HTTP server to catch the redirect
CODE=$(python3 -c "
import http.server, urllib.parse, sys
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        params = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query)
        print(params['code'][0], end='')
        self.send_response(200); self.end_headers()
        self.wfile.write(b'Auth complete! Close this tab.')
        sys.exit(0)
    def log_message(self, *a): pass
http.server.HTTPServer(('', 9876), H).handle_request()
")

# Exchange for tokens
curl -s -X POST https://oauth2.googleapis.com/token \
  -d "client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}&code=${CODE}&grant_type=authorization_code&redirect_uri=${REDIRECT_URI}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['refresh_token'])"
```

The `refresh_token` value printed is what you need. **Save it** -- you cannot retrieve it again.

### 4. Download the Binary

Download the latest binary for your platform from the [Releases page](https://github.com/ncosentino/google-keyword-planner-mcp/releases/latest).

| Platform | Go binary | C# AOT binary |
|----------|-----------|---------------|
| Linux x64 | `kwp-mcp-go-linux-amd64` | `kwp-mcp-csharp-linux-x64` |
| Linux ARM64 | `kwp-mcp-go-linux-arm64` | `kwp-mcp-csharp-linux-arm64` |
| macOS x64 | `kwp-mcp-go-darwin-amd64` | `kwp-mcp-csharp-osx-x64` |
| macOS ARM64 | `kwp-mcp-go-darwin-arm64` | `kwp-mcp-csharp-osx-arm64` |
| Windows x64 | `kwp-mcp-go-windows-amd64.exe` | `kwp-mcp-csharp-win-x64.exe` |
| Windows ARM64 | `kwp-mcp-go-windows-arm64.exe` | `kwp-mcp-csharp-win-arm64.exe` |

### 5. Configure Your MCP Client

#### GitHub Copilot CLI / Claude Code

```json
{
  "mcpServers": {
    "keyword-planner": {
      "type": "stdio",
      "command": "/path/to/kwp-mcp-go-linux-amd64",
      "args": [],
      "env": {
        "GOOGLE_ADS_DEVELOPER_TOKEN": "your-developer-token",
        "GOOGLE_ADS_CLIENT_ID": "your-client-id.apps.googleusercontent.com",
        "GOOGLE_ADS_CLIENT_SECRET": "your-client-secret",
        "GOOGLE_ADS_REFRESH_TOKEN": "your-refresh-token",
        "GOOGLE_ADS_CUSTOMER_ID": "your-sub-account-id",
        "GOOGLE_ADS_LOGIN_CUSTOMER_ID": "your-manager-account-id"
      }
    }
  }
}
```

> **Note:** Some MCP clients (e.g. GitHub Copilot CLI) require `"args": []` to be present when `"type": "stdio"` is specified. Claude Desktop does not require it. If your client fails to load the server, try adding `"args": []`.

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
        "GOOGLE_ADS_CUSTOMER_ID": "your-sub-account-id",
        "GOOGLE_ADS_LOGIN_CUSTOMER_ID": "your-manager-account-id"
      }
    }
  }
}
```

## Important: Manager Account + Sub-Account Setup

The `GOOGLE_ADS_LOGIN_CUSTOMER_ID` credential is the manager/MCC account ID. It is required **only when your `GOOGLE_ADS_CUSTOMER_ID` account is a sub-account managed through a manager account** -- it tells the API which manager to authenticate through by sending it as the `login-customer-id` HTTP header.

If the account you set as `GOOGLE_ADS_CUSTOMER_ID` is directly accessible by your OAuth credentials (i.e., you can see it in `listAccessibleCustomers` and it is not being accessed through a manager), you can omit `GOOGLE_ADS_LOGIN_CUSTOMER_ID`.

When in doubt: set it to your manager account ID. It doesn't hurt to include it even when not strictly required.

Both IDs can be found in the Google Ads UI -- the account number shown in the top-right corner when viewing that account. Dashes are optional -- `123-456-7890` and `1234567890` both work.

## Important: Billing Requirement

The Google Ads Keyword Planner API **requires an account with billing configured**. Accounts without an active payment method return errors regardless of credentials. You do not need to run any ads or spend money -- you just need a payment method on file.

## Common Errors

| Error | Meaning | Fix |
|-------|---------|-----|
| `DEVELOPER_TOKEN_NOT_APPROVED` | Your developer token is in test mode and cannot access real accounts | Apply for Basic access at `https://ads.google.com/aw/apicenter` and wait for Google approval (a few days) |
| `USER_PERMISSION_DENIED` with mention of `login-customer-id` | The account is a managed sub-account but `GOOGLE_ADS_LOGIN_CUSTOMER_ID` is missing or wrong | Set `GOOGLE_ADS_LOGIN_CUSTOMER_ID` to your manager account ID |
| `USER_PERMISSION_DENIED` without mention of `login-customer-id` | The OAuth user doesn't have access to the customer account | Complete the OAuth flow as the Google account that owns or has access to the ads account |
| HTTP 400 `INVALID_ARGUMENT` | Often a malformed customer ID or missing required field | Check that `GOOGLE_ADS_CUSTOMER_ID` contains only digits (dashes are stripped automatically) |

---

## Configuration Reference

Credentials are resolved in this priority order: **CLI flag > environment variable > `.env` file**.

| Credential | CLI flag | Environment variable | Required | Description |
|------------|----------|---------------------|----------|-------------|
| Developer token | `--developer-token` | `GOOGLE_ADS_DEVELOPER_TOKEN` | Yes | From Google Ads API Center (manager account) |
| OAuth2 client ID | `--client-id` | `GOOGLE_ADS_CLIENT_ID` | Yes | From GCP OAuth2 credentials |
| OAuth2 client secret | `--client-secret` | `GOOGLE_ADS_CLIENT_SECRET` | Yes | From GCP OAuth2 credentials |
| Refresh token | `--refresh-token` | `GOOGLE_ADS_REFRESH_TOKEN` | Yes | From one-time OAuth2 flow |
| Customer ID | `--customer-id` | `GOOGLE_ADS_CUSTOMER_ID` | Yes | Sub-account ID with billing (not the manager account) |
| Login customer ID | `--login-customer-id` | `GOOGLE_ADS_LOGIN_CUSTOMER_ID` | Yes* | Manager/MCC account ID. Required when using a sub-account. |

> \* Required when `GOOGLE_ADS_CUSTOMER_ID` is a managed sub-account (accessed through a manager/MCC account). Can be omitted if your OAuth credentials have direct access to the customer account.

### `.env` File

Place a `.env` file in the same directory as the binary:

```env
GOOGLE_ADS_DEVELOPER_TOKEN=your-developer-token
GOOGLE_ADS_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_ADS_CLIENT_SECRET=your-client-secret
GOOGLE_ADS_REFRESH_TOKEN=your-refresh-token
GOOGLE_ADS_CUSTOMER_ID=your-sub-account-id
GOOGLE_ADS_LOGIN_CUSTOMER_ID=your-manager-account-id
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

### Nick Cosentino -- Dev Leader

This MCP server was built by **[Nick Cosentino](https://www.devleader.ca)**, a software engineer and content creator known as **Dev Leader**. Nick creates practical .NET, C#, ASP.NET Core, Blazor, and software engineering content for intermediate to advanced developers -- covering everything from performance optimization and clean architecture to real-world career advice.

This tool was born out of real work researching keywords and planning SEO strategy for [devleader.ca](https://www.devleader.ca) and the desire to use AI assistants effectively during that process. It serves as a practical example of building Native AOT C# and idiomatic Go MCP servers with zero runtime dependencies.

**Find Nick online:**

- Blog: [https://www.devleader.ca](https://www.devleader.ca)
- YouTube: [https://www.youtube.com/@devleaderca](https://www.youtube.com/@devleaderca)
- Newsletter: [https://weekly.devleader.ca](https://weekly.devleader.ca)
- LinkedIn: [https://linkedin.com/in/nickcosentino](https://linkedin.com/in/nickcosentino)
- Linktree: [https://www.linktr.ee/devleader](https://www.linktr.ee/devleader)

### BrandGhost

[BrandGhost](https://www.brandghost.ai) is a social media automation platform built by Nick that lets content creators cross-post and schedule content across all social platforms in one click. If you create content and want to spend less time on distribution and more time creating, check it out.

---

## Contributing

Issues and pull requests welcome. Please run `go test ./...` and `dotnet test` before submitting.

---

## License

MIT License -- see [LICENSE](LICENSE).
