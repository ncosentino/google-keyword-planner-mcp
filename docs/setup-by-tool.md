---
description: Configure the Google Keyword Planner MCP server in GitHub Copilot CLI, Claude Desktop, Cursor, VS Code, Visual Studio, or via a .env file.
---

# Setup by Tool

Configuration snippets for each AI tool that supports MCP. In all cases, replace the path to the binary and provide your credentials via environment variables.

---

## GitHub Copilot CLI / Claude Code

Add to `.mcp.json` in your project or home directory:

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

!!! note
    Some MCP clients (including GitHub Copilot CLI) require `"args": []` when `"type": "stdio"` is specified. Claude Desktop does not require it. If your client fails to load the server, try adding `"args": []`.

---

## Claude Desktop

Add to `claude_desktop_config.json` (`~/Library/Application Support/Claude/` on macOS, `%APPDATA%\Claude\` on Windows):

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

---

## Cursor

Add to `.cursor/mcp.json` in your project root:

```json
{
  "mcpServers": {
    "keyword-planner": {
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

---

## VS Code

Add to `.vscode/mcp.json`:

```json
{
  "servers": {
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

---

## Visual Studio

Add to the MCP configuration in Visual Studio's GitHub Copilot settings:

```json
{
  "mcpServers": {
    "keyword-planner": {
      "type": "stdio",
      "command": "C:\\path\\to\\kwp-mcp-go-windows-amd64.exe",
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

---

## Using a .env File

Instead of putting credentials in your tool config, place a `.env` file in the same directory as the binary:

```env
GOOGLE_ADS_DEVELOPER_TOKEN=your-developer-token
GOOGLE_ADS_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_ADS_CLIENT_SECRET=your-client-secret
GOOGLE_ADS_REFRESH_TOKEN=your-refresh-token
GOOGLE_ADS_CUSTOMER_ID=your-sub-account-id
GOOGLE_ADS_LOGIN_CUSTOMER_ID=your-manager-account-id
```

Then the `env` block in your tool config can be omitted. The binary reads `.env` automatically from its working directory.

---

## Using CLI Arguments

You can also pass credentials directly on the command line:

```bash
./kwp-mcp-go-linux-amd64 \
  --developer-token your-token \
  --client-id your-client-id \
  --client-secret your-secret \
  --refresh-token your-refresh-token \
  --customer-id your-customer-id \
  --login-customer-id your-manager-id
```

See [Configuration](configuration.md) for the full resolution order.
