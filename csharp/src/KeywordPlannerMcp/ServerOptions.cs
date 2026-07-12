using System.Globalization;

namespace KeywordPlannerMcp;

/// <summary>Resolves command-line and environment configuration for the server host.</summary>
internal sealed record ServerOptions(
    string Transport,
    string ListenAddress,
    int Port,
    string? ShutdownToken)
{
    internal const string DefaultListenAddress = "127.0.0.1";
    internal const int DefaultPort = 8080;

    internal static bool IsHelpRequested(string[] args) =>
        args.Contains("--help", StringComparer.Ordinal) ||
        args.Contains("-h", StringComparer.Ordinal);

    internal static ServerOptions Parse(
        string[] args,
        Func<string, string?>? getEnvironmentVariable = null)
    {
        getEnvironmentVariable ??= Environment.GetEnvironmentVariable;

        var transport = GetOption(args, "--transport") ?? "stdio";
        if (transport is not ("stdio" or "http"))
        {
            throw new ArgumentException(
                $"Invalid transport \"{transport}\". Expected stdio or http.");
        }
        if (transport == "stdio")
        {
            return new ServerOptions(transport, DefaultListenAddress, DefaultPort, null);
        }

        var listenAddress = GetOption(args, "--listen-address")
            ?? getEnvironmentVariable("MCP_LISTEN_ADDRESS")
            ?? DefaultListenAddress;
        listenAddress = listenAddress.Trim();
        if (listenAddress.Length == 0)
        {
            throw new ArgumentException("The HTTP listen address must not be empty.");
        }

        var portValue = GetOption(args, "--port")
            ?? getEnvironmentVariable("PORT")
            ?? DefaultPort.ToString(CultureInfo.InvariantCulture);
        if (!int.TryParse(
            portValue,
            NumberStyles.None,
            CultureInfo.InvariantCulture,
            out var port) ||
            port is < 1 or > 65535)
        {
            throw new ArgumentException("The HTTP port must be an integer between 1 and 65535.");
        }

        return new ServerOptions(
            transport,
            listenAddress,
            port,
            getEnvironmentVariable("MCP_SHUTDOWN_TOKEN"));
    }

    internal static string Usage =>
        """
        Usage: kwp-mcp-csharp [options]

          --developer-token <token>     Google Ads developer token
          --client-id <id>              OAuth2 client ID
          --client-secret <secret>      OAuth2 client secret
          --refresh-token <token>       OAuth2 refresh token
          --customer-id <id>            Google Ads customer ID
          --login-customer-id <id>      Optional manager/MCC customer ID
          --transport stdio|http        Transport mode (default: stdio)
          --listen-address <address>    HTTP bind address (default: MCP_LISTEN_ADDRESS or 127.0.0.1)
          --port <port>                 HTTP port (default: PORT or 8080)
          --AllowedHosts <hosts>        Semicolon-separated HTTP Host allow-list
          -h, --help                    Show this help
        """;

    private static string? GetOption(string[] args, string name)
    {
        string? value = null;
        for (var index = 0; index < args.Length; index++)
        {
            var argument = args[index];
            if (argument.StartsWith(name + "=", StringComparison.Ordinal))
            {
                value = argument[(name.Length + 1)..];
                continue;
            }
            if (!string.Equals(argument, name, StringComparison.Ordinal))
            {
                continue;
            }
            if (index + 1 >= args.Length || args[index + 1].StartsWith("-", StringComparison.Ordinal))
            {
                throw new ArgumentException($"Missing value for {name}.");
            }
            value = args[++index];
        }
        return value;
    }
}
