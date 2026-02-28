# Getting Started

The Google Ads Keyword Planner API has more prerequisites than most APIs -- you need a Google Ads manager account, a developer token, and an OAuth2 refresh token. This page walks through every step.

---

## Prerequisites

You need all seven of these before the MCP server will work:

1. **Google Ads manager account (MCC)** -- developer tokens are only issued to manager accounts, not regular accounts. Create one free at [ads.google.com/home/tools/manager-accounts](https://ads.google.com/home/tools/manager-accounts/).

2. **Developer token with Basic or Standard access** -- in your manager account, go to `https://ads.google.com/aw/apicenter` and copy your developer token.

    !!! warning "New tokens start in test mode"
        A brand-new developer token can only call the API against [Google Ads test accounts](https://developers.google.com/google-ads/api/docs/best-practices/test-accounts). Calls to real accounts return `DEVELOPER_TOKEN_NOT_APPROVED` until you apply for Basic access. In the API Center, click **Apply for Basic Access** and fill in the form. Google reviews requests within a few days. Basic access is sufficient -- Standard access is not required.

3. **Google Ads account with billing configured** -- the Keyword Planner API requires an account with an active payment method. You do not need to run ads or spend money; you just need a payment method on file. This can be your manager account itself or a separate sub-account.

4. **Google Cloud project** with the [Google Ads API](https://console.cloud.google.com/apis/library/googleads.googleapis.com) enabled.

5. **OAuth2 client ID and client secret** from GCP (Desktop app type -- see GCP setup below).

6. **OAuth2 refresh token** obtained via a one-time authorization flow (see below). The OAuth2 flow must be completed as the Google account that has access to the Google Ads account you intend to use.

7. **Google Ads customer ID** -- the ID of the account with billing (found in the top-right of the Google Ads UI). Dashes are optional: `123-456-7890` and `1234567890` both work.

---

## Step 1: GCP Setup

1. Go to [GCP Console → APIs & Services → Credentials](https://console.cloud.google.com/apis/credentials)
2. Enable the **Google Ads API**: `https://console.cloud.google.com/apis/library/googleads.googleapis.com`
3. Configure the OAuth consent screen (`https://console.cloud.google.com/apis/credentials/consent`):
    - Select **External** → fill in app name and your email → **Save**
    - Add your own Google account as a **Test user**
    - Leave the app in **Testing** mode
4. Create an OAuth2 client: **+ Create Credentials → OAuth client ID → Desktop app** → Create
5. Copy the **Client ID** and **Client Secret**

---

## Step 2: Obtain a Refresh Token

Run this one-time flow to get a refresh token. It starts a local HTTP listener to automatically capture the authorization code:

=== "PowerShell (Windows)"

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

=== "bash (Linux/macOS)"

    ```bash
    CLIENT_ID="YOUR_CLIENT_ID"
    CLIENT_SECRET="YOUR_CLIENT_SECRET"
    REDIRECT_URI="http://localhost:9876"

    open "https://accounts.google.com/o/oauth2/v2/auth?client_id=${CLIENT_ID}&redirect_uri=${REDIRECT_URI}&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fadwords&access_type=offline&prompt=consent"

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

    curl -s -X POST https://oauth2.googleapis.com/token \
      -d "client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}&code=${CODE}&grant_type=authorization_code&redirect_uri=${REDIRECT_URI}" \
      | python3 -c "import sys,json; print(json.load(sys.stdin)['refresh_token'])"
    ```

!!! tip "Save your refresh token"
    The `refresh_token` value printed is what you need for `GOOGLE_ADS_REFRESH_TOKEN`. **Save it** -- you cannot retrieve it again from the API response.

---

## Step 3: Download a Binary

Go to the [Releases page](https://github.com/ncosentino/google-keyword-planner-mcp/releases/latest) and download the binary for your platform:

| Platform | Go binary | C# binary |
|----------|-----------|-----------|
| Linux x64 | `kwp-mcp-go-linux-amd64` | `kwp-mcp-csharp-linux-x64` |
| Linux arm64 | `kwp-mcp-go-linux-arm64` | `kwp-mcp-csharp-linux-arm64` |
| macOS x64 (Intel) | `kwp-mcp-go-darwin-amd64` | `kwp-mcp-csharp-osx-x64` |
| macOS arm64 (Apple Silicon) | `kwp-mcp-go-darwin-arm64` | `kwp-mcp-csharp-osx-arm64` |
| Windows x64 | `kwp-mcp-go-windows-amd64.exe` | `kwp-mcp-csharp-win-x64.exe` |
| Windows arm64 | `kwp-mcp-go-windows-arm64.exe` | `kwp-mcp-csharp-win-arm64.exe` |

See [Go vs C#](implementations.md) if you're unsure which to pick. Either works.

---

## Step 4: Configure Your AI Tool

See **[Setup by Tool](setup-by-tool.md)** for exact configuration snippets for Claude, Claude Desktop, GitHub Copilot, Cursor, VS Code, and Visual Studio.

The minimum required environment variables are:

```env
GOOGLE_ADS_DEVELOPER_TOKEN=your-developer-token
GOOGLE_ADS_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_ADS_CLIENT_SECRET=your-client-secret
GOOGLE_ADS_REFRESH_TOKEN=your-refresh-token
GOOGLE_ADS_CUSTOMER_ID=your-sub-account-id
GOOGLE_ADS_LOGIN_CUSTOMER_ID=your-manager-account-id
```

!!! note "About GOOGLE_ADS_LOGIN_CUSTOMER_ID"
    This is your manager/MCC account ID -- required when `GOOGLE_ADS_CUSTOMER_ID` is a sub-account accessed through a manager. If your OAuth credentials have direct access to the customer account (not through a manager), you can omit it. When in doubt, include it -- it doesn't hurt.

---

## Next Steps

- [Setup by Tool](setup-by-tool.md) -- exact JSON config for each AI tool
- [Configuration](configuration.md) -- full credential reference
- [Troubleshooting](troubleshooting.md) -- common errors and fixes
