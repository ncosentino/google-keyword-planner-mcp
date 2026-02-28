---
description: Zero-dependency MCP server exposing Google Ads Keyword Planner to AI assistants. Pre-built binaries for Linux, macOS, and Windows. No runtime required.
---

# Google Keyword Planner MCP

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

## Quick Overview

Three MCP tools are exposed:

| Tool | What it does |
|------|-------------|
| [`generate_keyword_ideas`](tools/generate-keyword-ideas/) | Related keywords with search volume and CPC from seed keywords or a URL |
| [`get_historical_metrics`](tools/get-historical-metrics/) | Historical search volume, competition, and CPC for a list of keywords |
| [`get_keyword_forecast`](tools/get-keyword-forecast/) | Projected impressions, clicks, and cost at a given max CPC bid |

---

## Get Started

**[→ Getting Started](getting-started/)** -- prerequisites, OAuth2 setup, binary download, and configuration.

---

## About

Built by **[Nick Cosentino](https://www.devleader.ca)** (Dev Leader) -- a software engineer and content creator covering .NET, C#, and software architecture. Available in both Go and C# (Native AOT) with zero runtime dependencies.

- Blog: [devleader.ca](https://www.devleader.ca)
- GitHub: [ncosentino/google-keyword-planner-mcp](https://github.com/ncosentino/google-keyword-planner-mcp)
