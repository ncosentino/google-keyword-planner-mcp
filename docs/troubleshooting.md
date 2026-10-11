---
description: Fix DEVELOPER_TOKEN_NOT_APPROVED, USER_PERMISSION_DENIED, billing errors, and other common issues with the Google Keyword Planner MCP server.
---

# Troubleshooting

## Common Errors

| Error | Meaning | Fix |
|-------|---------|-----|
| `CLOUD_PROJECT_NOT_APPROVED_FOR_PRODUCTION` | The OAuth Google Cloud project has Test access only | Apply for Explorer or a higher access level in Google Cloud Console |
| Legacy `DEVELOPER_TOKEN_NOT_APPROVED` or `ACTION_NOT_PERMITTED` | API access is not approved for the requested account/environment | Review project access levels in the Google Ads API overview, not the retired API Center |
| `USER_PERMISSION_DENIED` (mentions `login-customer-id`) | The account is a managed sub-account but `GOOGLE_ADS_LOGIN_CUSTOMER_ID` is missing or set to the wrong value | Set `GOOGLE_ADS_LOGIN_CUSTOMER_ID` to your manager account ID |
| `USER_PERMISSION_DENIED` (no mention of `login-customer-id`) | The Google account used in the OAuth flow does not have access to the ads account | Re-run the OAuth refresh token flow as the Google account that owns or has access to the ads account |
| HTTP 400 `INVALID_ARGUMENT` | Often a malformed customer ID or a missing required field | Check that `GOOGLE_ADS_CUSTOMER_ID` contains only digits -- dashes are stripped automatically, but extra characters are not |

---

## Billing Requirement

The Google Ads Keyword Planner API **requires an account with billing configured**. Accounts without an active payment method return errors regardless of how valid the credentials are.

You do **not** need to run any ads or spend money -- you just need a payment method on file.

- To add billing: go to **Billing & Payments** in the Google Ads UI for the account you plan to use.
- The account can be the manager account or a sub-account -- either works as long as it has billing and is the one you point `GOOGLE_ADS_CUSTOMER_ID` at.

---

## Manager Account Setup

If you receive permission errors and are unsure whether you need `GOOGLE_ADS_LOGIN_CUSTOMER_ID`:

1. Check whether the account in `GOOGLE_ADS_CUSTOMER_ID` is a **standalone account** or a **managed sub-account**.
   - In the Google Ads UI, accounts accessed through a manager show a breadcrumb at the top (e.g., `My Manager > Sub Account`).
2. If it is a sub-account, set `GOOGLE_ADS_LOGIN_CUSTOMER_ID` to the manager account ID.
3. If it is a standalone account, you can omit `GOOGLE_ADS_LOGIN_CUSTOMER_ID` -- but including it with the correct manager ID never hurts.

---

## Test Access Projects

Projects with **Test access** cannot query production accounts:

- v25 production calls return `CLOUD_PROJECT_NOT_APPROVED_FOR_PRODUCTION`.
- Calls to [Google Ads test accounts](https://developers.google.com/google-ads/api/docs/best-practices/test-accounts) work correctly.

Apply for Explorer or a higher access level through the
[Google Ads API overview](https://console.cloud.google.com/google/ads-apis/overview).
Developer tokens were sunset on September 9, 2026 and are now optional ignored
headers. See Google's [migration guide](https://developers.google.com/google-ads/api/docs/api-policy/developer-token).
