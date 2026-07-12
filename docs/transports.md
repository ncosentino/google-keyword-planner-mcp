---
description: Run Google Keyword Planner MCP over STDIO or shared Streamable HTTP.
---

# Transports

Both native implementations support STDIO and stateless Streamable HTTP.

## STDIO

STDIO remains the default. The MCP client launches one subprocess for each
client session.

```bash
./kwp-mcp-go-linux-amd64
```

Credentials resolve from CLI flags, environment variables, or `.env`.

## Streamable HTTP

```bash
./kwp-mcp-go-linux-amd64 \
  --transport http \
  --listen-address 127.0.0.1 \
  --port 8082
```

The C# binary accepts the same transport, address, and port arguments.

| Endpoint | Purpose |
|---|---|
| `/mcp` | Stateless Streamable HTTP MCP |
| `/health` | Health, service identity, and version metadata |
| `/shutdown` | Optional manager-authenticated loopback shutdown |

`--listen-address` falls back to `MCP_LISTEN_ADDRESS` and then `127.0.0.1`.
`--port` falls back to `PORT` and then `8080`.

Configure an HTTP-capable client with:

```json
{
  "mcpServers": {
    "keyword-planner": {
      "type": "http",
      "url": "http://127.0.0.1:8082/mcp",
      "tools": ["*"]
    }
  }
}
```

Credentials belong to the server process, not the HTTP client configuration.

## Security defaults

- The default listener is loopback-only.
- Go and C# validate Host headers.
- Go and C# reject mismatched browser Origin headers.
- MCP request bodies are limited to 1 MiB.
- Header and keep-alive timeouts are bounded.
- The transport is stateless and requires no session affinity.

Go uses the comma-separated `--allowed-hosts` option. C# uses standard ASP.NET
Core `AllowedHosts`, whose default is:

```text
localhost;127.0.0.1;[::1]
```

The built-in host does not authenticate MCP callers. Non-loopback deployments
require TLS, authentication and authorization, trusted proxy configuration,
and ingress limits.

See [Shared Service](shared-service.md) for cross-session lifecycle management.
