// Package keywordplanner provides types for the Google Ads Keyword Planner API.
package keywordplanner

// KeywordIdea is a keyword suggestion with historical performance metrics.
type KeywordIdea struct {
	Text                   string          `json:"text"`
	AvgMonthlySearches     int64           `json:"avgMonthlySearches"`
	Competition            string          `json:"competition"`
	CompetitionIndex       int32           `json:"competitionIndex"`
	LowTopOfPageBidMicros  int64           `json:"lowTopOfPageBidMicros,omitempty"`
	HighTopOfPageBidMicros int64           `json:"highTopOfPageBidMicros,omitempty"`
	MonthlySearchVolumes   []MonthlyVolume `json:"monthlySearchVolumes,omitempty"`
}

// IdeaOptions filters and shapes the ideas returned by GenerateKeywordIdeas.
// The zero value returns every idea without monthly volumes.
type IdeaOptions struct {
	// Limit caps the number of ideas returned; 0 means no cap.
	Limit int
	// MinMonthlySearches drops ideas with a lower average monthly search volume.
	MinMonthlySearches int64
	// IncludeMonthlyVolumes adds the per-month search volumes to each idea.
	IncludeMonthlyVolumes bool
}

// KeywordIdeasResponse is the result of generating keyword ideas.
type KeywordIdeasResponse struct {
	SeedKeywords []string      `json:"seedKeywords,omitempty"`
	URL          string        `json:"url,omitempty"`
	Ideas        []KeywordIdea `json:"ideas"`
	Count        int           `json:"count"`
	// TotalFound is the number of ideas the API returned before filtering and truncation.
	TotalFound int `json:"totalFound"`
	// Language and GeoTargetConstants echo the targeting the request actually used.
	Language           string   `json:"language,omitempty"`
	GeoTargetConstants []string `json:"geoTargetConstants,omitempty"`
}

// KeywordMetrics holds historical search metrics for a single keyword.
type KeywordMetrics struct {
	Text                   string          `json:"text"`
	AvgMonthlySearches     int64           `json:"avgMonthlySearches"`
	Competition            string          `json:"competition"`
	CompetitionIndex       int32           `json:"competitionIndex"`
	LowTopOfPageBidMicros  int64           `json:"lowTopOfPageBidMicros,omitempty"`
	HighTopOfPageBidMicros int64           `json:"highTopOfPageBidMicros,omitempty"`
	MonthlySearchVolumes   []MonthlyVolume `json:"monthlySearchVolumes,omitempty"`
}

// MonthlyVolume is the search volume for a specific month.
type MonthlyVolume struct {
	Year            int32 `json:"year"`
	Month           int32 `json:"month"`
	MonthlySearches int64 `json:"monthlySearches"`
}

// HistoricalMetricsResponse is the result of a historical metrics lookup.
type HistoricalMetricsResponse struct {
	Keywords []KeywordMetrics `json:"keywords"`
	Count    int              `json:"count"`
}

// KeywordForecastMetrics holds projected performance for a keyword.
//
// Impressions, click-through rate and conversion rate were removed from
// KeywordForecastMetrics in Google Ads API v24/v25 and are no longer returned
// by generateKeywordForecastMetrics, so they are not modelled here.
type KeywordForecastMetrics struct {
	Text             string   `json:"text,omitempty"`
	Clicks           *float64 `json:"clicks,omitempty"`
	CostMicros       *int64   `json:"costMicros,omitempty"`
	AverageCPCMicros *int64   `json:"averageCpcMicros,omitempty"`
}

// ForecastResponse is the result of a keyword forecast request.
//
// Keywords holds one entry per requested keyword, each forecast on its own —
// the API only ever returns a campaign-level aggregate per request, so a
// per-keyword breakdown requires one request per keyword. Total is the
// aggregate for all keywords forecast together in a single ad group, which is
// what the campaign would actually deliver.
type ForecastResponse struct {
	Keywords           []KeywordForecastMetrics `json:"keywords"`
	Total              KeywordForecastMetrics   `json:"total"`
	ForecastDays       int                      `json:"forecastDays"`
	MaxCPCMicros       int64                    `json:"maxCpcMicros"`
	StartDate          string                   `json:"startDate"`
	EndDate            string                   `json:"endDate"`
	GeoTargetConstants []string                 `json:"geoTargetConstants,omitempty"`
	Language           string                   `json:"language,omitempty"`
	CurrencyCode       string                   `json:"currencyCode,omitempty"`
	MatchType          string                   `json:"matchType"`
}

// --- Google Ads API raw request/response types ---

type generateKeywordIdeasRequest struct {
	CustomerID         string             `json:"customerId,omitempty"`
	Language           string             `json:"language,omitempty"`
	GeoTargetConstants []string           `json:"geoTargetConstants,omitempty"`
	KeywordPlanNetwork string             `json:"keywordPlanNetwork,omitempty"`
	KeywordSeed        *keywordSeed       `json:"keywordSeed,omitempty"`
	URLSeed            *urlSeed           `json:"urlSeed,omitempty"`
	KeywordAndURLSeed  *keywordAndURLSeed `json:"keywordAndUrlSeed,omitempty"`
	PageToken          string             `json:"pageToken,omitempty"`
}

type keywordSeed struct {
	Keywords []string `json:"keywords"`
}

type urlSeed struct {
	URL string `json:"url"`
}

type keywordAndURLSeed struct {
	URL      string   `json:"url"`
	Keywords []string `json:"keywords"`
}

type generateKeywordIdeasResponse struct {
	Results       []keywordIdeaResult `json:"results"`
	NextPageToken string              `json:"nextPageToken"`
}

type keywordIdeaResult struct {
	Text               string             `json:"text"`
	KeywordIdeaMetrics keywordIdeaMetrics `json:"keywordIdeaMetrics"`
}

type keywordIdeaMetrics struct {
	AvgMonthlySearches     string                `json:"avgMonthlySearches"`
	Competition            string                `json:"competition"`
	CompetitionIndex       string                `json:"competitionIndex"`
	LowTopOfPageBidMicros  string                `json:"lowTopOfPageBidMicros"`
	HighTopOfPageBidMicros string                `json:"highTopOfPageBidMicros"`
	MonthlySearchVolumes   []monthlySearchVolume `json:"monthlySearchVolumes"`
}

type generateHistoricalMetricsRequest struct {
	Keywords           []string `json:"keywords"`
	Language           string   `json:"language,omitempty"`
	GeoTargetConstants []string `json:"geoTargetConstants,omitempty"`
	KeywordPlanNetwork string   `json:"keywordPlanNetwork,omitempty"`
}

// generateHistoricalMetricsResponse mirrors
// GenerateKeywordHistoricalMetricsResponse, whose results live under "results".
// Parsing a "metrics" field instead silently yielded an empty keyword list.
type generateHistoricalMetricsResponse struct {
	Results []historicalMetricsResult `json:"results"`
}

type historicalMetricsResult struct {
	Text           string            `json:"text"`
	KeywordMetrics historicalMetrics `json:"keywordMetrics"`
}

// historicalMetrics mirrors KeywordPlanHistoricalMetrics. Every int64 field is
// serialised as a JSON string by the REST API, so they are decoded as strings
// and converted afterwards — decoding them as numbers fails the whole response.
type historicalMetrics struct {
	AvgMonthlySearches     string                `json:"avgMonthlySearches"`
	Competition            string                `json:"competition"`
	CompetitionIndex       string                `json:"competitionIndex"`
	LowTopOfPageBidMicros  string                `json:"lowTopOfPageBidMicros"`
	HighTopOfPageBidMicros string                `json:"highTopOfPageBidMicros"`
	MonthlySearchVolumes   []monthlySearchVolume `json:"monthlySearchVolumes"`
}

type monthlySearchVolume struct {
	Year            string `json:"year"`
	Month           string `json:"month"`
	MonthlySearches string `json:"monthlySearches"`
}

// generateForecastMetricsRequest mirrors GenerateKeywordForecastMetricsRequest.
// The forecast period is a top-level DateRange and the campaign is a
// CampaignToForecast — the older shape that nested dates inside a
// "campaignForecastSpec" object was never a valid field name.
type generateForecastMetricsRequest struct {
	CurrencyCode   string           `json:"currencyCode,omitempty"`
	ForecastPeriod dateRange        `json:"forecastPeriod"`
	Campaign       campaignForecast `json:"campaign"`
}

type dateRange struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type campaignForecast struct {
	LanguageConstants  []string          `json:"languageConstants,omitempty"`
	GeoTargetConstants []string          `json:"geoTargetConstants,omitempty"`
	BiddingStrategy    biddingStrategy   `json:"biddingStrategy"`
	AdGroups           []adGroupForecast `json:"adGroups"`
}

type biddingStrategy struct {
	ManualCpcBiddingStrategy manualCpcBiddingStrategy `json:"manualCpcBiddingStrategy"`
}

type manualCpcBiddingStrategy struct {
	MaxCPCBidMicros string `json:"maxCpcBidMicros"`
}

// adGroupForecast mirrors ForecastAdGroup. In v25 the ad group carries a plain
// list of KeywordInfo; the v23 "biddableKeywords" / "negativeKeywords" /
// "maxCpcBidMicros" fields were removed.
type adGroupForecast struct {
	Keywords []forecastKeyword `json:"keywords"`
}

type forecastKeyword struct {
	Text      string `json:"text"`
	MatchType string `json:"matchType"`
}

// generateForecastMetricsResponse mirrors GenerateKeywordForecastMetricsResponse,
// which carries a single campaign-level aggregate and no per-keyword breakdown.
type generateForecastMetricsResponse struct {
	CampaignForecastMetrics *forecastMetricData `json:"campaignForecastMetrics"`
}

type forecastMetricData struct {
	Clicks           *float64 `json:"clicks"`
	CostMicros       *int64   `json:"costMicros,string"`
	AverageCPCMicros *int64   `json:"averageCpcMicros,string"`
}
