// Package keywordplanner provides a client for the Google Ads Keyword Planner API.
package keywordplanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

const (
	tokenURL      = "https://oauth2.googleapis.com/token"
	adsAPIBase    = "https://googleads.googleapis.com/v25"
	adsAPIVersion = "v25"
	httpTimeout   = 30 * time.Second

	// forecastConcurrency caps the parallel requests of one keyword forecast.
	forecastConcurrency = 5
)

// Client calls the Google Ads Keyword Planner API.
type Client struct {
	httpClient      *http.Client
	developerToken  string
	customerID      string
	loginCustomerID string
	baseURL         string
	tokenSource     oauth2.TokenSource
	// defaultLanguage and defaultGeoTargets apply to calls that do not set their own.
	defaultLanguage   string
	defaultGeoTargets []string
	planningLimiter   interface{ Wait(context.Context) error }
	lookup            constantLookup
}

// SetDefaultTargeting sets the language and geo targets used by calls that leave
// them empty. Both accept the forms ResolveLanguage and ResolveGeoTargets accept.
// Empty defaults mean no restriction.
func (c *Client) SetDefaultTargeting(language string, geoTargets []string) {
	c.defaultLanguage = language
	c.defaultGeoTargets = geoTargets
}

func (c *Client) targeting(language string, geoTargets []string) (string, []string, error) {
	if language == "" {
		language = c.defaultLanguage
	}
	if geoTargets == nil {
		geoTargets = c.defaultGeoTargets
	}
	if language != "" && strings.TrimSpace(language) == "" {
		return "", nil, fmt.Errorf("language must not be blank; use 'all' for all languages")
	}
	language = strings.TrimSpace(language)
	if strings.EqualFold(language, "all") {
		language = ""
	}
	targets := make([]string, 0, len(geoTargets))
	seen := make(map[string]bool)
	for _, target := range geoTargets {
		target = strings.TrimSpace(target)
		if target == "" {
			return "", nil, fmt.Errorf("geo targets must not be blank; use [] for worldwide targeting")
		}
		if !seen[target] {
			targets = append(targets, target)
			seen[target] = true
		}
	}
	if len(targets) > 10 {
		return "", nil, fmt.Errorf("at most 10 geo targets are supported")
	}
	return language, targets, nil
}

// NewClient creates a Client with the provided OAuth2 credentials.
// loginCustomerID is the manager/MCC account ID; set it when customerID is a sub-account.
func NewClient(developerToken, clientID, clientSecret, refreshToken, customerID, loginCustomerID string) *Client {
	return NewClientWithBaseURL(developerToken, clientID, clientSecret, refreshToken, customerID, loginCustomerID, adsAPIBase)
}

// NewClientWithBaseURL creates a Client with a custom API base URL. Intended for testing.
func NewClientWithBaseURL(developerToken, clientID, clientSecret, refreshToken, customerID, loginCustomerID, baseURL string) *Client {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: tokenURL},
	}
	token := &oauth2.Token{RefreshToken: refreshToken}
	ts := conf.TokenSource(context.Background(), token)

	base := oauth2.NewClient(context.Background(), ts)
	base.Timeout = httpTimeout

	return &Client{
		httpClient:      base,
		developerToken:  developerToken,
		customerID:      customerID,
		loginCustomerID: loginCustomerID,
		baseURL:         baseURL,
		tokenSource:     ts,
		planningLimiter: rate.NewLimiter(rate.Every(time.Second), 1),
	}
}

// newTestClient creates a Client that uses a plain http.Client (no OAuth2) for unit tests.
func newTestClient(developerToken, customerID, loginCustomerID, baseURL string, httpClient *http.Client) *Client {
	return &Client{
		httpClient:      httpClient,
		developerToken:  developerToken,
		customerID:      customerID,
		loginCustomerID: loginCustomerID,
		baseURL:         baseURL,
	}
}

// NewTestClient is exported solely for use in package-level tests.
// Do not use in production code.
func NewTestClient(developerToken, customerID, loginCustomerID, baseURL string, httpClient *http.Client) *Client {
	return newTestClient(developerToken, customerID, loginCustomerID, baseURL, httpClient)
}

// GenerateKeywordIdeas returns keyword ideas for the given seed keywords and/or URL.
// geoTargetConstants scopes volumes to specific locations (e.g.
// "geoTargetConstants/2036" for Australia). When empty, the API applies no
// location filter and returns worldwide aggregate volumes — for a
// country-specific term that can differ from the local figure by an order of
// magnitude, with nothing in the response indicating which you received.
//
// keywordPlanNetwork is "GOOGLE_SEARCH" or "GOOGLE_SEARCH_AND_PARTNERS"; when
// empty the API default (search and partners) applies, which reads higher than
// the search-only figure the Keyword Planner web UI shows by default.
//
// Ideas are returned sorted by average monthly searches, highest first; opts
// filters and truncates that list.
func (c *Client) GenerateKeywordIdeas(
	ctx context.Context,
	seedKeywords []string,
	seedURL string,
	language string,
	geoTargetConstants []string,
	keywordPlanNetwork string,
	opts IdeaOptions,
) (*KeywordIdeasResponse, error) {
	language, geoTargetConstants, err := c.resolveTargeting(ctx, language, geoTargetConstants)
	if err != nil {
		return nil, err
	}
	reqBody := c.buildKeywordIdeasRequest(seedKeywords, seedURL, language, geoTargetConstants, keywordPlanNetwork)
	endpoint := fmt.Sprintf("%s/customers/%s:generateKeywordIdeas", c.baseURL, c.customerID)

	var raw generateKeywordIdeasResponse
	seenPages := make(map[string]bool)
	for {
		var page generateKeywordIdeasResponse
		if err := c.post(ctx, endpoint, reqBody, &page); err != nil {
			return nil, err
		}
		raw.Results = append(raw.Results, page.Results...)
		if page.NextPageToken == "" {
			break
		}
		if seenPages[page.NextPageToken] {
			return nil, fmt.Errorf("keyword ideas API repeated a pagination token")
		}
		seenPages[page.NextPageToken] = true
		reqBody.PageToken = page.NextPageToken
	}

	ideas := make([]KeywordIdea, 0, len(raw.Results))
	for _, r := range raw.Results {
		idea := KeywordIdea{
			Text:                   r.Text,
			AvgMonthlySearches:     parseI64(r.KeywordIdeaMetrics.AvgMonthlySearches),
			Competition:            r.KeywordIdeaMetrics.Competition,
			CompetitionIndex:       int32(parseI64(r.KeywordIdeaMetrics.CompetitionIndex)),
			LowTopOfPageBidMicros:  parseI64(r.KeywordIdeaMetrics.LowTopOfPageBidMicros),
			HighTopOfPageBidMicros: parseI64(r.KeywordIdeaMetrics.HighTopOfPageBidMicros),
		}
		if idea.AvgMonthlySearches < opts.MinMonthlySearches {
			continue
		}
		if opts.IncludeMonthlyVolumes {
			idea.MonthlySearchVolumes = toMonthlyVolumes(r.KeywordIdeaMetrics.MonthlySearchVolumes)
		}
		ideas = append(ideas, idea)
	}
	sort.SliceStable(ideas, func(i, j int) bool {
		return ideas[i].AvgMonthlySearches > ideas[j].AvgMonthlySearches
	})
	if opts.Limit > 0 && len(ideas) > opts.Limit {
		ideas = ideas[:opts.Limit]
	}

	return &KeywordIdeasResponse{
		SeedKeywords:       seedKeywords,
		URL:                seedURL,
		Ideas:              ideas,
		Count:              len(ideas),
		TotalFound:         len(raw.Results),
		Language:           language,
		GeoTargetConstants: geoTargetConstants,
	}, nil
}

// GetHistoricalMetrics returns historical search metrics for a list of keywords.
//
// language and geoTargetConstants are optional but strongly recommended:
// without them the search volumes cover the whole world in every language,
// which is not what a targeted campaign will see.
// keywordPlanNetwork is optional; see GenerateKeywordIdeas.
func (c *Client) GetHistoricalMetrics(
	ctx context.Context,
	keywords []string,
	language string,
	geoTargetConstants []string,
	keywordPlanNetwork string,
) (*HistoricalMetricsResponse, error) {
	language, geoTargetConstants, err := c.resolveTargeting(ctx, language, geoTargetConstants)
	if err != nil {
		return nil, err
	}
	reqBody := generateHistoricalMetricsRequest{
		Keywords:           keywords,
		Language:           language,
		GeoTargetConstants: geoTargetConstants,
		KeywordPlanNetwork: keywordPlanNetwork,
	}
	endpoint := fmt.Sprintf("%s/customers/%s:generateKeywordHistoricalMetrics", c.baseURL, c.customerID)

	var raw generateHistoricalMetricsResponse
	if err := c.post(ctx, endpoint, reqBody, &raw); err != nil {
		return nil, err
	}

	metrics := make([]KeywordMetrics, 0, len(raw.Results))
	for _, r := range raw.Results {
		metrics = append(metrics, KeywordMetrics{
			Text:                   r.Text,
			AvgMonthlySearches:     parseI64(r.KeywordMetrics.AvgMonthlySearches),
			Competition:            r.KeywordMetrics.Competition,
			CompetitionIndex:       int32(parseI64(r.KeywordMetrics.CompetitionIndex)),
			LowTopOfPageBidMicros:  parseI64(r.KeywordMetrics.LowTopOfPageBidMicros),
			HighTopOfPageBidMicros: parseI64(r.KeywordMetrics.HighTopOfPageBidMicros),
			MonthlySearchVolumes:   toMonthlyVolumes(r.KeywordMetrics.MonthlySearchVolumes),
		})
	}

	return &HistoricalMetricsResponse{Keywords: metrics, Count: len(metrics)}, nil
}

// ForecastRequest describes a keyword forecast to run.
//
// GeoTargetConstants and Language are optional but strongly recommended: with
// neither set the API forecasts worldwide traffic across all languages, which
// bears no resemblance to what a geographically targeted campaign will deliver.
type ForecastRequest struct {
	Keywords                []string
	MatchType               string
	MaxCPCMicros            int64
	ForecastDays            int
	GeoTargetConstants      []string
	Language                string
	CurrencyCode            string
	IncludeKeywordBreakdown bool
}

// GetKeywordForecast returns projected performance metrics for a set of keywords.
//
// The API returns one campaign-level aggregate per request, so this issues one
// request per keyword to build the per-keyword breakdown, plus one request with
// every keyword in a single ad group for the campaign total. The requests run
// concurrently; a single keyword needs only one. The per-keyword
// figures will not sum to the total: keywords in one ad group compete for the
// same impressions and the total accounts for that overlap.
func (c *Client) GetKeywordForecast(ctx context.Context, req ForecastRequest) (*ForecastResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(req.Keywords) == 0 {
		return nil, fmt.Errorf("at least one keyword is required")
	}
	var err error
	req.Language, req.GeoTargetConstants, err = c.resolveTargeting(ctx, req.Language, req.GeoTargetConstants)
	if err != nil {
		return nil, err
	}
	forecastDays := req.ForecastDays
	if forecastDays <= 0 {
		forecastDays = 30
	}
	maxCPCMicros := req.MaxCPCMicros
	if maxCPCMicros <= 0 {
		maxCPCMicros = 1_000_000
	}
	matchType := strings.ToUpper(strings.TrimSpace(req.MatchType))
	if matchType == "" {
		matchType = "BROAD"
	}
	if matchType != "BROAD" && matchType != "PHRASE" && matchType != "EXACT" {
		return nil, fmt.Errorf("match_type must be BROAD, PHRASE or EXACT")
	}
	if forecastDays > 364 {
		return nil, fmt.Errorf("forecast_days must not exceed 364")
	}

	// The forecast period must start after today in the account's time zone.
	// UTC tomorrow is still "today" for accounts east of UTC around midnight,
	// so start two days out.
	period := forecastPeriodFor(time.Now(), forecastDays)
	startDate, endDate := period.StartDate, period.EndDate

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	forecast := func(keywords []string) (KeywordForecastMetrics, error) {
		infos := make([]forecastKeyword, 0, len(keywords))
		for _, kw := range keywords {
			infos = append(infos, forecastKeyword{Text: kw, MatchType: matchType})
		}

		reqBody := generateForecastMetricsRequest{
			CurrencyCode:   req.CurrencyCode,
			ForecastPeriod: dateRange{StartDate: startDate, EndDate: endDate},
			Campaign: campaignForecast{
				GeoTargetConstants: req.GeoTargetConstants,
				BiddingStrategy: biddingStrategy{
					ManualCpcBiddingStrategy: manualCpcBiddingStrategy{
						MaxCPCBidMicros: strconv.FormatInt(maxCPCMicros, 10),
					},
				},
				AdGroups: []adGroupForecast{{Keywords: infos}},
			},
		}
		if req.Language != "" {
			reqBody.Campaign.LanguageConstants = []string{req.Language}
		}

		endpoint := fmt.Sprintf("%s/customers/%s:generateKeywordForecastMetrics", c.baseURL, c.customerID)

		var raw generateForecastMetricsResponse
		if err := c.post(ctx, endpoint, reqBody, &raw); err != nil {
			return KeywordForecastMetrics{}, err
		}

		m := raw.CampaignForecastMetrics
		if m == nil {
			return KeywordForecastMetrics{}, fmt.Errorf("forecast response is missing campaignForecastMetrics")
		}
		return KeywordForecastMetrics{
			Clicks:           m.Clicks,
			CostMicros:       m.CostMicros,
			AverageCPCMicros: m.AverageCPCMicros,
		}, nil
	}

	perKeyword := make([]KeywordForecastMetrics, 0, len(req.Keywords))
	if req.IncludeKeywordBreakdown {
		perKeyword = make([]KeywordForecastMetrics, len(req.Keywords))
	}
	// The requests are independent, so run them concurrently. The first failure
	// cancels the rest and is the error reported.
	var (
		total    KeywordForecastMetrics
		wg       sync.WaitGroup
		slots    = make(chan struct{}, forecastConcurrency)
		failOnce sync.Once
		firstErr error
	)
	run := func(job func() error) bool {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			return false
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			if ctx.Err() != nil {
				return
			}
			if err := job(); err != nil {
				failOnce.Do(func() {
					firstErr = err
					cancel()
				})
			}
		}()
		return true
	}

	if req.IncludeKeywordBreakdown {
		for i, kw := range req.Keywords {
			if !run(func() error {
				metrics, err := forecast([]string{kw})
				if err != nil {
					return fmt.Errorf("forecasting %q: %w", kw, err)
				}
				metrics.Text = kw
				perKeyword[i] = metrics
				return nil
			}) {
				break
			}
		}
	}
	// A single keyword is its own ad group, so its forecast is the total.
	if !req.IncludeKeywordBreakdown || len(req.Keywords) > 1 {
		run(func() error {
			metrics, err := forecast(req.Keywords)
			if err != nil {
				return fmt.Errorf("forecasting combined ad group: %w", err)
			}
			total = metrics
			return nil
		})
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.IncludeKeywordBreakdown && len(req.Keywords) == 1 {
		total = perKeyword[0]
		total.Text = ""
	}

	return &ForecastResponse{
		Keywords:           perKeyword,
		Total:              total,
		ForecastDays:       forecastDays,
		MaxCPCMicros:       maxCPCMicros,
		StartDate:          startDate,
		EndDate:            endDate,
		GeoTargetConstants: req.GeoTargetConstants,
		Language:           req.Language,
		CurrencyCode:       req.CurrencyCode,
		MatchType:          matchType,
	}, nil
}

func forecastPeriodFor(now time.Time, days int) dateRange {
	start := now.UTC().AddDate(0, 0, 2)
	return dateRange{
		StartDate: start.Format("2006-01-02"),
		EndDate:   start.AddDate(0, 0, days-1).Format("2006-01-02"),
	}
}

func (c *Client) post(ctx context.Context, endpoint string, body, out any) error {
	if c.planningLimiter != nil &&
		(strings.HasSuffix(endpoint, ":generateKeywordIdeas") ||
			strings.HasSuffix(endpoint, ":generateKeywordHistoricalMetrics") ||
			strings.HasSuffix(endpoint, ":generateKeywordForecastMetrics")) {
		if err := c.planningLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("waiting for planning quota: %w", err)
		}
	}
	reqBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshalling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBytes))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.developerToken != "" {
		req.Header.Set("developer-token", c.developerToken)
	}
	if c.loginCustomerID != "" {
		req.Header.Set("login-customer-id", c.loginCustomerID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Google Ads API returned HTTP %d: %s",
			resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	return nil
}

func (c *Client) buildKeywordIdeasRequest(
	seedKeywords []string,
	seedURL, language string,
	geoTargetConstants []string,
	keywordPlanNetwork string,
) generateKeywordIdeasRequest {
	req := generateKeywordIdeasRequest{
		Language:           language,
		GeoTargetConstants: geoTargetConstants,
		KeywordPlanNetwork: keywordPlanNetwork,
	}
	switch {
	case len(seedKeywords) > 0 && seedURL != "":
		req.KeywordAndURLSeed = &keywordAndURLSeed{URL: seedURL, Keywords: seedKeywords}
	case seedURL != "":
		req.URLSeed = &urlSeed{URL: seedURL}
	default:
		req.KeywordSeed = &keywordSeed{Keywords: seedKeywords}
	}
	return req
}

func toMonthlyVolumes(raw []monthlySearchVolume) []MonthlyVolume {
	monthly := make([]MonthlyVolume, 0, len(raw))
	for _, m := range raw {
		monthly = append(monthly, MonthlyVolume{
			Year:            int32(parseI64(m.Year)),
			Month:           parseMonthEnum(m.Month),
			MonthlySearches: parseI64(m.MonthlySearches),
		})
	}
	return monthly
}

func parseI64(s string) int64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// parseMonthEnum converts "JANUARY" → 1, etc.
func parseMonthEnum(month string) int32 {
	months := map[string]int32{
		"JANUARY": 1, "FEBRUARY": 2, "MARCH": 3, "APRIL": 4,
		"MAY": 5, "JUNE": 6, "JULY": 7, "AUGUST": 8,
		"SEPTEMBER": 9, "OCTOBER": 10, "NOVEMBER": 11, "DECEMBER": 12,
	}
	if v, ok := months[strings.ToUpper(month)]; ok {
		return v
	}
	return 0
}
