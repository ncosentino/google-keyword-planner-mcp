using ModelContextProtocol.Client;
using ModelContextProtocol.Protocol;
using Xunit;

namespace KeywordPlannerMcp.Tests;

/// <summary>
/// Characterization test for the default (stdio) transport path, written ahead
/// of the ModelContextProtocol SDK dependency modernization (issue #11).
/// </summary>
/// <remarks>
/// Program.cs's stdio branch (AddMcpServer().WithStdioServerTransport()) binds
/// directly to the real, process-wide Console.In/Console.Out, and the SDK
/// offers no overload that accepts a substitute stream (unlike the Go SDK's
/// mcp.IOTransport, which accepts an arbitrary Reader/Writer). So the only
/// faithful way to exercise this path is to spawn the real compiled server as
/// a subprocess and connect a real MCP client over its actual stdin/stdout --
/// the same way Claude Desktop, Claude Code, and other stdio-based MCP clients
/// launch this server in production. Before this test, nothing automated
/// exercised the stdio code path at all: every other test used the HTTP
/// transport (HostingHttpTests) or called tool methods directly
/// (KeywordPlannerToolTests).
/// </remarks>
public sealed class StdioTransportTests
{
    [Fact]
    public async Task StdioTransport_ServesRealSession_ListsToolsAndReturnsValidationError()
    {
        // Locate the real compiled server DLL via a type it defines, rather than
        // hardcoding a relative path that would be fragile across Debug/Release
        // and TFM-specific output folders.
        var serverDllPath = typeof(Hosting).Assembly.Location;

        await using var client = await McpClient.CreateAsync(new StdioClientTransport(new StdioClientTransportOptions
        {
            Name = "kwp-mcp-stdio-test",
            Command = "dotnet",
            Arguments =
            [
                serverDllPath,
                "--developer-token", "dev-token",
                "--client-id", "client-id",
                "--client-secret", "client-secret",
                "--refresh-token", "refresh-token",
                "--customer-id", "3778350596",
            ],
        }));

        var tools = await client.ListToolsAsync();

        Assert.Equal(3, tools.Count);
        Assert.Contains(tools, t => t.Name == "generate_keyword_ideas");
        Assert.Contains(tools, t => t.Name == "get_historical_metrics");
        Assert.Contains(tools, t => t.Name == "get_keyword_forecast");

        // No seedKeywords or url: this must hit the validation short-circuit in
        // KeywordPlannerTool rather than actually calling the Google Ads API (which
        // would fail anyway, since the credentials above aren't real).
        var result = await client.CallToolAsync("generate_keyword_ideas", new Dictionary<string, object?>());

        var text = Assert.IsType<TextContentBlock>(Assert.Single(result.Content));
        Assert.Contains("At least one of seedKeywords or url must be provided", text.Text, StringComparison.Ordinal);
    }
}
