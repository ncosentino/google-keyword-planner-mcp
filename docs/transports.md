---
description: Run the Google Keyword Planner MCP server over HTTP instead of stdio -- flags, environment variables, security defaults, and example client configuration for both the Go and C# binaries.
---

# Transports

Both binaries default to **stdio** (the standard MCP transport for locally-launched servers) and support an **HTTP** transport for remote/networked deployments. Behavior is equivalent between Go and C#, with one cosmetic difference noted below.

---

## stdio (default)

No flags needed -- this is the default, and matches every example elsewhere in these docs. Your AI tool launches the binary as a subprocess and communicates over its stdin/stdout.

---

## HTTP

Pass `--transport http` to serve MCP over the [Streamable HTTP](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports#streamable-http) transport instead:

=== "Go"

    ```bash
    ./kwp-mcp-go-linux-amd64 --transport http \
      --developer-token your-token \
      --client-id your-client-id \
      --client-secret your-secret \
      --refresh-token your-refresh-token \
      --customer-id your-customer-id
    ```

=== "C#"

    ```bash
    ./kwp-mcp-csharp-linux-x64 --transport http \
      --developer-token your-token \
      --client-id your-client-id \
      --client-secret your-secret \
      --refresh-token your-refresh-token \
      --customer-id your-customer-id
    ```

The server listens on the `PORT` environment variable (default `8080`), matching the convention several hosting platforms inject automatically:

```bash
PORT=9000 ./kwp-mcp-go-linux-amd64 --transport http --developer-token ... 
```

Credentials still resolve the same way as stdio (CLI flag > environment variable > `.env` file -- see [Configuration](configuration.md)); only the transport itself changes.

### Connecting a client

Point an MCP client that supports HTTP transports at the server's URL instead of launching it as a subprocess:

```json
{
  "mcpServers": {
    "keyword-planner": {
      "type": "http",
      "url": "http://localhost:8080/"
    }
  }
}
```

Credentials are still supplied to the *server* process (via its own CLI flags/env vars/`.env` file) -- an HTTP client config has no `env` block, since the server isn't being launched by the client.

### Session mode

Both implementations run in **stateless** mode: no session affinity, no in-memory session state kept between requests. This is the documented recommendation for tool servers with no need to send requests back to the client, and it keeps horizontal scaling and restarts simple.

---

## Security: Host header allow-list

Neither Go's `net/http` nor Kestrel validate the `Host` header by default, which would otherwise leave an HTTP-transport deployment open to [DNS rebinding](https://en.wikipedia.org/wiki/DNS_rebinding) -- a browser reaching your local server through an attacker-controlled DNS name that resolves back to it. Both binaries default to a loopback-only allow-list (`localhost`, `127.0.0.1`, `[::1]`) and reject anything else before the request reaches the MCP handler.

To allow additional hosts (e.g. behind a reverse proxy with a real domain):

=== "Go"

    Use `--allowed-hosts`, a comma-separated list:

    ```bash
    ./kwp-mcp-go-linux-amd64 --transport http --allowed-hosts "mcp.example.com,localhost" ...
    ```

    A disallowed `Host` header gets `403 Forbidden`.

=== "C#"

    Set the standard ASP.NET Core `AllowedHosts` configuration key, a semicolon-separated list. Any standard .NET configuration source works, including a plain command-line argument:

    ```bash
    ./kwp-mcp-csharp-linux-x64 --transport http --AllowedHosts "mcp.example.com;localhost" ...
    ```

    A disallowed `Host` header gets `400 Bad Request` -- this is ASP.NET Core's built-in Host Filtering Middleware, not custom logic, so it uses that middleware's standard response.

!!! warning "Don't disable this in production"
    If you put either binary behind a reverse proxy or expose it beyond your own machine, keep the allow-list scoped to the exact host names you expect. Only add a wildcard-like setup if you understand the DNS rebinding risk it reopens.

---

## What HTTP transport does *not* include

This is a transport flag, not a hosting product. Neither binary bundles:

- A Dockerfile or container image
- Cloud-provider-specific deployment automation
- Authentication/authorization in front of the MCP endpoint itself (beyond the Host allow-list above)

If you deploy either binary behind HTTP, you're responsible for TLS termination, authentication, and network exposure -- typically via a reverse proxy or your hosting platform's own ingress layer.
