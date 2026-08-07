// Package keywordplanner provides types for the Google Ads Keyword Planner API.
package keywordplanner

// KeywordIdea is a keyword suggestion with historical performance metrics.
type KeywordIdea struct {
	Text               string `json:"text"`
	AvgMonthlySearches int64  `json:"avgMonthlySearches"`
	Competition        string `json:"competition"`
	LowTopOfPageBidMicros  int64 `json:"lowTopOfPageBidMicros,omitempty"`
	HighTopOfPageBidMicros int64 `json:"highTopOfPageBidMicros,omitempty"`
}

// KeywordIdeasResponse is the result of generating keyword ideas.
type KeywordIdeasResponse struct {
	SeedKeywords []string      `json:"seedKeywords,omitempty"`
	URL          string        `json:"url,omitempty"`
	Ideas        []KeywordIdea `json:"ideas"`
	Count        int           `json:"count"`
}

// KeywordMetrics holds historical search metrics for a single keyword.
type KeywordMetrics struct {
	Text               string          `json:"text"`
	AvgMonthlySearches int64           `json:"avgMonthlySearches"`
	Competition        string          `json:"competition"`
	CompetitionIndex   int32           `json:"competitionIndex"`
	LowTopOfPageBidMicros  int64       `json:"lowTopOfPageBidMicros,omitempty"`
	HighTopOfPageBidMicros int64       `json:"highTopOfPageBidMicros,omitempty"`
	MonthlySearchVolumes []MonthlyVolume `json:"monthlySearchVolumes,omitempty"`
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

// KeywordForecastMetrics holds projected campaign-level performance for a set of keywords.
type KeywordForecastMetrics struct {
	Text        string  `json:"text"`
	Impressions float64 `json:"impressions"`
	Clicks      float64 `json:"clicks"`
	CostMicros  int64   `json:"costMicros"`
	CTR         float64 `json:"ctr"`
}

// ForecastResponse is the result of a keyword forecast request.
type ForecastResponse struct {
	Keywords     []KeywordForecastMetrics `json:"keywords"`
	ForecastDays int                      `json:"forecastDays"`
	MaxCPCMicros int64                    `json:"maxCpcMicros"`
}

// --- Google Ads API raw request/response types ---

type generateKeywordIdeasRequest struct {
	CustomerID         string             `json:"customerId,omitempty"`
	Language           string             `json:"language,omitempty"`
	GeoTargetConstants []string           `json:"geoTargetConstants,omitempty"`
	KeywordSeed        *keywordSeed       `json:"keywordSeed,omitempty"`
	URLSeed            *urlSeed           `json:"urlSeed,omitempty"`
	KeywordAndURLSeed  *keywordAndURLSeed `json:"keywordAndUrlSeed,omitempty"`
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
	Results []keywordIdeaResult `json:"results"`
}

type keywordIdeaResult struct {
	Text               string             `json:"text"`
	KeywordIdeaMetrics keywordIdeaMetrics `json:"keywordIdeaMetrics"`
}

type keywordIdeaMetrics struct {
	AvgMonthlySearches     string `json:"avgMonthlySearches"`
	Competition            string `json:"competition"`
	CompetitionIndex       string `json:"competitionIndex"`
	LowTopOfPageBidMicros  string `json:"lowTopOfPageBidMicros"`
	HighTopOfPageBidMicros string `json:"highTopOfPageBidMicros"`
}

type generateHistoricalMetricsRequest struct {
	Keywords []string `json:"keywords"`
}

// The API returns the top-level key "results", not "metrics".
type generateHistoricalMetricsResponse struct {
	Metrics []historicalMetricsResult `json:"results"`
}

type historicalMetricsResult struct {
	Text           string            `json:"text"`
	KeywordMetrics historicalMetrics `json:"keywordMetrics"`
}

// All numeric fields are returned as strings by the Google Ads API.
type historicalMetrics struct {
	AvgMonthlySearches     string                `json:"avgMonthlySearches"`
	Competition            string                `json:"competition"`
	CompetitionIndex       string                `json:"competitionIndex"`
	LowTopOfPageBidMicros  string                `json:"lowTopOfPageBidMicros"`
	HighTopOfPageBidMicros string                `json:"highTopOfPageBidMicros"`
	MonthlySearchVolumes   []monthlySearchVolume `json:"monthlySearchVolumes"`
}

// All fields are returned as strings by the Google Ads API.
type monthlySearchVolume struct {
	Year            string `json:"year"`
	Month           string `json:"month"`
	MonthlySearches string `json:"monthlySearches"`
}

// The Google Ads API generateKeywordForecastMetrics endpoint expects "campaign"
// as the top-level key (not "campaignForecastSpec"), with "forecastPeriod" as a
// separate sibling field. The start date must be tomorrow or later.
type generateForecastMetricsRequest struct {
	Campaign       campaignToForecast `json:"campaign"`
	ForecastPeriod forecastPeriod     `json:"forecastPeriod"`
}

type campaignToForecast struct {
	KeywordPlanNetwork string            `json:"keywordPlanNetwork"`
	BiddingStrategy    biddingStrategy   `json:"biddingStrategy"`
	AdGroups           []adGroupForecast `json:"adGroups"`
}

type forecastPeriod struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type biddingStrategy struct {
	ManualCpcBiddingStrategy manualCpcBiddingStrategy `json:"manualCpcBiddingStrategy"`
}

type manualCpcBiddingStrategy struct {
	MaxCPCBidMicros string `json:"maxCpcBidMicros"`
}

type adGroupForecast struct {
	Biddable []adGroupForecastKeyword `json:"biddableKeywords"`
}

type adGroupForecastKeyword struct {
	Keyword forecastKeyword `json:"keyword"`
}

type forecastKeyword struct {
	Text      string `json:"text"`
	MatchType string `json:"matchType"`
}

// The API returns campaign-level aggregate metrics under "campaignForecastMetrics",
// not per-keyword data under "adGroupForecastMetrics".
type generateForecastMetricsResponse struct {
	CampaignForecastMetrics campaignForecastMetrics `json:"campaignForecastMetrics"`
}

type campaignForecastMetrics struct {
	Impressions      float64 `json:"impressions"`
	Clicks           float64 `json:"clicks"`
	CostMicros       string  `json:"costMicros"`
	ClickThroughRate float64 `json:"clickThroughRate"`
	AverageCpcMicros string  `json:"averageCpcMicros"`
	Conversions      float64 `json:"conversions"`
}
