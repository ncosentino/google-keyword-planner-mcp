---
description: Run one Google Keyword Planner MCP service shared by every agent session.
---

# Running One Shared Service

Streamable HTTP lets every local agent connect to one long-lived Keyword
Planner process instead of launching one STDIO child per session.

## Prepare credentials

Place the Google Ads credentials in a `.env` file beside the binary:

```env
GOOGLE_ADS_DEVELOPER_TOKEN=your-developer-token
GOOGLE_ADS_CLIENT_ID=your-client-id
GOOGLE_ADS_CLIENT_SECRET=your-client-secret
GOOGLE_ADS_REFRESH_TOKEN=your-refresh-token
GOOGLE_ADS_CUSTOMER_ID=your-customer-id
GOOGLE_ADS_LOGIN_CUSTOMER_ID=your-manager-id
```

Protect this file so only the service account running the process can read it.

## Start the service

```bash
./kwp-mcp-go-linux-amd64 \
  --transport http \
  --listen-address 127.0.0.1 \
  --port 8082
```

The C# Native AOT binary uses the same arguments.

## Lifecycle management

Use a platform service supervisor, or reuse the generic
[`manage-mcp-service.ps1`](https://github.com/ncosentino/google-psi-mcp/blob/main/scripts/manage-mcp-service.ps1)
maintained for the native NexusLabs MCP servers:

```powershell
.\manage-mcp-service.ps1 Start `
  -ServiceName google-keyword-planner-mcp `
  -BinaryPath C:\path\to\kwp-mcp-go.exe `
  -Port 8082
```

The manager health-checks before starting, serializes concurrent start
attempts, records process identity, and uses a per-run authenticated loopback
shutdown before falling back to terminating the verified process.

## Configure Copilot CLI

```json
{
  "mcpServers": {
    "google-keyword-planner": {
      "type": "http",
      "url": "http://127.0.0.1:8082/mcp",
      "tools": ["*"]
    }
  }
}
```

Remove `command`, `args`, and `env` from the HTTP entry because Copilot no
longer launches the process. Existing sessions retain their STDIO child until
they restart.

## Network deployment

The generic manager is deliberately intended for loopback services.
Non-loopback hosting requires a platform supervisor, TLS, authentication and
authorization on every request, trusted proxy handling, and ingress limits.
