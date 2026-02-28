---
description: Full reference for all Google Keyword Planner MCP credentials -- CLI flags, environment variables, .env file, customer ID formats, and developer token access levels.
---

# Configuration

Credentials are resolved in this priority order: **CLI flag > environment variable > `.env` file**.

## Credential Reference

| Credential | CLI flag | Environment variable | Required | Description |
|------------|----------|---------------------|----------|-------------|
| Developer token | `--developer-token` | `GOOGLE_ADS_DEVELOPER_TOKEN` | Yes | From Google Ads API Center (manager account only) |
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

## Developer Token Access Levels

| Access level | Can call real accounts? | How to get |
|---|---|---|
| Test mode (default) | ❌ No -- only [test accounts](https://developers.google.com/google-ads/api/docs/best-practices/test-accounts) | Issued automatically when you create a developer token |
| Basic access | ✅ Yes | Click **Apply for Basic Access** at `https://ads.google.com/aw/apicenter` and wait a few days |
| Standard access | ✅ Yes | Apply separately; not required for Keyword Planner |
