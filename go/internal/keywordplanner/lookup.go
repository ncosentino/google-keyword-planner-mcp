package keywordplanner

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const (
	languagePrefix  = "languageConstants/"
	geoTargetPrefix = "geoTargetConstants/"

	languageQuery = "SELECT language_constant.id, language_constant.code, language_constant.name " +
		"FROM language_constant"
	countryQuery = "SELECT geo_target_constant.id, geo_target_constant.country_code, geo_target_constant.name " +
		"FROM geo_target_constant " +
		"WHERE geo_target_constant.target_type = 'Country' AND geo_target_constant.status = 'ENABLED'"
)

// constantLookup caches the language and country tables fetched from the API.
// A table is cached only once it has loaded successfully, so a transient
// failure is retried on the next call.
type constantLookup struct {
	mu        sync.Mutex
	languages map[string]string
	countries map[string]string
}

// GeoTarget is a location that can be passed as a geo target.
type GeoTarget struct {
	ResourceName  string `json:"resourceName"`
	Name          string `json:"name"`
	CanonicalName string `json:"canonicalName,omitempty"`
	CountryCode   string `json:"countryCode,omitempty"`
	TargetType    string `json:"targetType,omitempty"`
	Reach         int64  `json:"reach,omitempty"`
}

// GeoTargetSearchResponse is the result of a geo target search.
type GeoTargetSearchResponse struct {
	GeoTargets []GeoTarget `json:"geoTargets"`
	Count      int         `json:"count"`
}

// ResolveLanguage turns a language given as a resource name
// ("languageConstants/1001"), a bare ID ("1001"), an ISO code ("de") or an
// English name ("German") into a resource name. An empty input stays empty.
func (c *Client) ResolveLanguage(ctx context.Context, language string) (string, error) {
	if language != "" && strings.TrimSpace(language) == "" {
		return "", fmt.Errorf("language must not be blank")
	}
	language = strings.TrimSpace(language)
	if language == "" || strings.HasPrefix(language, languagePrefix) {
		return language, nil
	}
	if isDigits(language) {
		return languagePrefix + language, nil
	}

	table, err := c.languageTable(ctx)
	if err != nil {
		return "", fmt.Errorf("looking up language %q: %w", language, err)
	}
	if id, ok := table[strings.ToLower(language)]; ok {
		return languagePrefix + id, nil
	}
	return "", fmt.Errorf("unknown language %q: use an ISO code such as 'de' or a resource name such as 'languageConstants/1001'", language)
}

// ResolveGeoTargets turns each geo target given as a resource name
// ("geoTargetConstants/2040"), a bare ID ("2040"), an ISO country code ("AT")
// or an English country name ("Austria") into a resource name. Regions and
// cities have no code; find their resource names with SearchGeoTargets.
func (c *Client) ResolveGeoTargets(ctx context.Context, geoTargets []string) ([]string, error) {
	if len(geoTargets) == 0 {
		return geoTargets, nil
	}
	resolved := make([]string, 0, len(geoTargets))
	for _, target := range geoTargets {
		target = strings.TrimSpace(target)
		switch {
		case target == "":
			return nil, fmt.Errorf("geo targets must not be blank; use [] for worldwide targeting")
		case strings.HasPrefix(target, geoTargetPrefix):
			resolved = append(resolved, target)
		case isDigits(target):
			resolved = append(resolved, geoTargetPrefix+target)
		default:
			table, err := c.countryTable(ctx)
			if err != nil {
				return nil, fmt.Errorf("looking up geo target %q: %w", target, err)
			}
			id, ok := table[strings.ToLower(target)]
			if !ok {
				return nil, fmt.Errorf("unknown geo target %q: use an ISO country code such as 'AT', or find regions and cities with search_geo_targets", target)
			}
			resolved = append(resolved, geoTargetPrefix+id)
		}
	}
	unique := make([]string, 0, len(resolved))
	seen := make(map[string]bool)
	for _, target := range resolved {
		if !seen[target] {
			unique = append(unique, target)
			seen[target] = true
		}
	}
	return unique, nil
}

// SearchGeoTargets finds locations (countries, regions, cities) by name.
// countryCode optionally restricts the search to one country; locale sets the
// language of the returned names and defaults to English.
func (c *Client) SearchGeoTargets(ctx context.Context, names []string, countryCode, locale string) (*GeoTargetSearchResponse, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("at least one location name is required")
	}
	if locale == "" {
		locale = "en"
	}
	reqBody := suggestGeoTargetsRequest{
		Locale:        locale,
		CountryCode:   strings.ToUpper(strings.TrimSpace(countryCode)),
		LocationNames: locationNames{Names: names},
	}
	endpoint := fmt.Sprintf("%s/geoTargetConstants:suggest", c.baseURL)

	var raw suggestGeoTargetsResponse
	if err := c.post(ctx, endpoint, reqBody, &raw); err != nil {
		return nil, err
	}

	targets := make([]GeoTarget, 0, len(raw.Suggestions))
	for _, s := range raw.Suggestions {
		targets = append(targets, GeoTarget{
			ResourceName:  s.GeoTargetConstant.ResourceName,
			Name:          s.GeoTargetConstant.Name,
			CanonicalName: s.GeoTargetConstant.CanonicalName,
			CountryCode:   s.GeoTargetConstant.CountryCode,
			TargetType:    s.GeoTargetConstant.TargetType,
			Reach:         parseI64(s.Reach),
		})
	}
	return &GeoTargetSearchResponse{GeoTargets: targets, Count: len(targets)}, nil
}

// resolveTargeting applies the client defaults to an unset language or geo
// target list and resolves codes and names to resource names.
func (c *Client) resolveTargeting(ctx context.Context, language string, geoTargets []string) (string, []string, error) {
	language, geoTargets, err := c.targeting(language, geoTargets)
	if err != nil {
		return "", nil, err
	}
	language, err = c.ResolveLanguage(ctx, language)
	if err != nil {
		return "", nil, err
	}
	geoTargets, err = c.ResolveGeoTargets(ctx, geoTargets)
	if err != nil {
		return "", nil, err
	}
	return language, geoTargets, nil
}

func (c *Client) languageTable(ctx context.Context) (map[string]string, error) {
	c.lookup.mu.Lock()
	defer c.lookup.mu.Unlock()
	if c.lookup.languages != nil {
		return c.lookup.languages, nil
	}

	var raw searchResponse
	if err := c.post(ctx, c.searchEndpoint(), searchRequest{Query: languageQuery}, &raw); err != nil {
		return nil, err
	}
	table := make(map[string]string, 2*len(raw.Results))
	for _, r := range raw.Results {
		addLookupKey(table, r.LanguageConstant.Code, r.LanguageConstant.ID)
		addLookupKey(table, r.LanguageConstant.Name, r.LanguageConstant.ID)
	}
	c.lookup.languages = table
	return table, nil
}

func (c *Client) countryTable(ctx context.Context) (map[string]string, error) {
	c.lookup.mu.Lock()
	defer c.lookup.mu.Unlock()
	if c.lookup.countries != nil {
		return c.lookup.countries, nil
	}

	var raw searchResponse
	if err := c.post(ctx, c.searchEndpoint(), searchRequest{Query: countryQuery}, &raw); err != nil {
		return nil, err
	}
	table := make(map[string]string, 2*len(raw.Results))
	for _, r := range raw.Results {
		addLookupKey(table, r.GeoTargetConstant.CountryCode, r.GeoTargetConstant.ID)
		addLookupKey(table, r.GeoTargetConstant.Name, r.GeoTargetConstant.ID)
	}
	c.lookup.countries = table
	return table, nil
}

func (c *Client) searchEndpoint() string {
	return fmt.Sprintf("%s/customers/%s/googleAds:search", c.baseURL, c.customerID)
}

func addLookupKey(table map[string]string, key, id string) {
	if key = strings.ToLower(strings.TrimSpace(key)); key != "" && id != "" {
		table[key] = id
	}
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// --- Google Ads API raw request/response types ---

type searchRequest struct {
	Query string `json:"query"`
}

type searchResponse struct {
	Results []searchRow `json:"results"`
}

type searchRow struct {
	LanguageConstant  languageConstant  `json:"languageConstant"`
	GeoTargetConstant geoTargetConstant `json:"geoTargetConstant"`
}

type languageConstant struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type geoTargetConstant struct {
	ResourceName  string `json:"resourceName"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	CountryCode   string `json:"countryCode"`
	TargetType    string `json:"targetType"`
	CanonicalName string `json:"canonicalName"`
}

type suggestGeoTargetsRequest struct {
	Locale        string        `json:"locale,omitempty"`
	CountryCode   string        `json:"countryCode,omitempty"`
	LocationNames locationNames `json:"locationNames"`
}

type locationNames struct {
	Names []string `json:"names"`
}

type suggestGeoTargetsResponse struct {
	Suggestions []geoTargetSuggestion `json:"geoTargetConstantSuggestions"`
}

type geoTargetSuggestion struct {
	GeoTargetConstant geoTargetConstant `json:"geoTargetConstant"`
	Reach             string            `json:"reach"`
}
