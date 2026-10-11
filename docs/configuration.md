---
description: Credential flags, environment variables, .env files, customer IDs, and Google Cloud project access levels.
---

# Configuration

Credentials are resolved in this priority order: **CLI flag > environment variable > `.env` file**.

## Credential Reference

| Credential | CLI flag | Environment variable | Required | Description |
|------------|----------|---------------------|----------|-------------|
| Legacy developer token | `--developer-token` | `GOOGLE_ADS_DEVELOPER_TOKEN` | No | Retained for compatibility; ignored by Google since September 9, 2026 |
| OAuth2 client ID | `--client-id` | `GOOGLE_ADS_CLIENT_ID` | Yes | From GCP OAuth2 credentials (Desktop app type) |
| OAuth2 client secret | `--client-secret` | `GOOGLE_ADS_CLIENT_SECRET` | Yes | From GCP OAuth2 credentials |
| Refresh token | `--refresh-token` | `GOOGLE_ADS_REFRESH_TOKEN` | Yes | From one-time OAuth2 flow (see [Getting Started](getting-started.md)) |
| Customer ID | `--customer-id` | `GOOGLE_ADS_CUSTOMER_ID` | Yes | Sub-account ID with billing -- dashes are stripped automatically |
| Login customer ID | `--login-customer-id` | `GOOGLE_ADS_LOGIN_CUSTOMER_ID` | Conditional | Manager/MCC account ID -- required when using a sub-account |

!!! note "When is GOOGLE_ADS_LOGIN_CUSTOMER_ID required?"
    It is required when `GOOGLE_ADS_CUSTOMER_ID` is a managed sub-account accessed through a manager/MCC account. It tells the API which manager to authenticate through by sending it as the `login-customer-id` HTTP header.

    If the OAuth user has direct access to the customer account (not through a manager), you can omit it. When in doubt, include it -- it doesn't hurt to set it even when not strictly required.

---

## .env File

Place a `.env` file in the same directory as the binary. The binary reads it automatically:

```env
GOOGLE_ADS_DEVELOPER_TOKEN=your-developer-token
GOOGLE_ADS_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_ADS_CLIENT_SECRET=your-client-secret
GOOGLE_ADS_REFRESH_TOKEN=your-refresh-token
GOOGLE_ADS_CUSTOMER_ID=your-sub-account-id
GOOGLE_ADS_LOGIN_CUSTOMER_ID=your-manager-account-id
```

---

## Customer ID Format

Both IDs can be found in the Google Ads UI -- the account number shown in the top-right corner when viewing that account:

| Format | Example | Accepted? |
|--------|---------|-----------|
| With dashes | `123-456-7890` | ✅ Yes |
| Digits only | `1234567890` | ✅ Yes |

Dashes are stripped automatically before the API call.

---

## Google Cloud Project Access Levels

| Access level | Can call real accounts? | How to get |
|---|---|---|
| Test access | ❌ No -- only [test accounts](https://developers.google.com/google-ads/api/docs/best-practices/test-accounts) | Google Cloud project test access |
| Explorer access | ✅ Yes | Apply through the Google Ads API overview in Google Cloud Console |
| Basic access | ✅ Yes | Apply through Google Cloud Console; brand verification is required for new applications |
| Standard access | ✅ Yes | Apply separately; not required for Keyword Planner |

Developer tokens were sunset on September 9, 2026. Access is determined by the
Google Cloud project used for OAuth credentials. Do not use the retired Ads
API Center signup process. See Google's [migration guide](https://developers.google.com/google-ads/api/docs/api-policy/developer-token).

---

## Transport

Credentials resolve the same way regardless of transport. See
**[Transports](transports.md)** for running one shared HTTP service.

## HTTP host

```bash
./kwp-mcp-go-linux-amd64 \
  --transport http \
  --listen-address 127.0.0.1 \
  --port 8082
```

- `--listen-address` overrides `MCP_LISTEN_ADDRESS`; the default is `127.0.0.1`.
- `--port` overrides `PORT`; the default is `8080`.
- MCP is served at `/mcp`.
- Health metadata is served at `/health`.

Go accepts a comma-separated `--allowed-hosts` list. C# uses the standard
semicolon-separated ASP.NET Core `AllowedHosts` setting.
