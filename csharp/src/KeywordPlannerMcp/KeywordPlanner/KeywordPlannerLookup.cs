namespace KeywordPlannerMcp.KeywordPlanner;

internal sealed partial class KeywordPlannerClient
{
    private const string LanguageQuery =
        "SELECT language_constant.id, language_constant.code, language_constant.name FROM language_constant";
    private const string CountryQuery =
        "SELECT geo_target_constant.id, geo_target_constant.country_code, geo_target_constant.name FROM geo_target_constant WHERE geo_target_constant.target_type = 'Country' AND geo_target_constant.status = 'ENABLED'";
    private readonly SemaphoreSlim _languageGate = new(1, 1);
    private readonly SemaphoreSlim _countryGate = new(1, 1);
    private Dictionary<string, string>? _languages;
    private Dictionary<string, string>? _countries;

    private async Task<string> ResolveConstantAsync(string value, bool language, CancellationToken cancellationToken)
    {
        var prefix = language ? "languageConstants/" : "geoTargetConstants/";
        if (value.StartsWith(prefix, StringComparison.Ordinal)) return value;
        if (value.All(c => c is >= '0' and <= '9')) return prefix + value;
        var gate = language ? _languageGate : _countryGate;
        await gate.WaitAsync(cancellationToken);
        try
        {
            var table = language ? _languages : _countries;
            if (table is null)
            {
                table = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
                var request = new ConstantSearchRequest(language ? LanguageQuery : CountryQuery);
                var seenTokens = new HashSet<string>(StringComparer.Ordinal);
                while (true)
                {
                    var response = await PostAsync("googleAds:search", request,
                        KwpJsonContext.Default.ConstantSearchRequest,
                        KwpJsonContext.Default.ConstantSearchResponse, cancellationToken);
                    foreach (var row in response.Results)
                    {
                        var id = language ? row.LanguageConstant.Id : row.GeoTargetConstant.Id;
                        var code = language ? row.LanguageConstant.Code : row.GeoTargetConstant.CountryCode;
                        var name = language ? row.LanguageConstant.Name : row.GeoTargetConstant.Name;
                        if (string.IsNullOrEmpty(id)) continue;
                        if (!string.IsNullOrWhiteSpace(code)) table[code.Trim()] = id;
                        if (!string.IsNullOrWhiteSpace(name)) table[name.Trim()] = id;
                    }
                    if (string.IsNullOrEmpty(response.NextPageToken)) break;
                    if (!seenTokens.Add(response.NextPageToken))
                        throw new InvalidOperationException("Constant search API repeated a pagination token.");
                    request = request with { PageToken = response.NextPageToken };
                }
                if (language) _languages = table;
                else _countries = table;
            }
            return table.TryGetValue(value, out var resourceId) ? prefix + resourceId
                : throw new ArgumentException($"Unknown {(language ? "language" : "geo target")} '{value}'. Use an ISO code, English name, or resource name.");
        }
        finally { gate.Release(); }
    }

    internal async Task<GeoTargetSearchResponse> SearchGeoTargetsAsync(
        IReadOnlyList<string> names, string? countryCode = null, string? locale = null,
        CancellationToken cancellationToken = default)
    {
        if (names.Count == 0 || names.Any(string.IsNullOrWhiteSpace))
            throw new ArgumentException("At least one nonblank location name is required.");
        var request = new SuggestGeoTargetsRequest(
            string.IsNullOrWhiteSpace(locale) ? "en" : locale.Trim(),
            string.IsNullOrWhiteSpace(countryCode) ? null : countryCode.Trim().ToUpperInvariant(),
            new LocationNames(names.Select(n => n.Trim()).ToArray()));
        var response = await PostAsync("geoTargetConstants:suggest", request,
            KwpJsonContext.Default.SuggestGeoTargetsRequest,
            KwpJsonContext.Default.SuggestGeoTargetsResponse, cancellationToken);
        var targets = response.GeoTargetConstantSuggestions.Select(s => new GeoTarget(
            s.GeoTargetConstant.ResourceName, s.GeoTargetConstant.Name,
            s.GeoTargetConstant.CanonicalName, s.GeoTargetConstant.CountryCode,
            s.GeoTargetConstant.TargetType, s.Reach)).ToList();
        return new GeoTargetSearchResponse(targets, targets.Count);
    }
}
