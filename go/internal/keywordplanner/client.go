// Package keywordplanner provides a client for the Google Ads Keyword Planner API.
package keywordplanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	tokenURL    = "https://oauth2.googleapis.com/token"
	adsAPIBase  = "https://googleads.googleapis.com/v23"
	adsAPIVersion = "v23"
	httpTimeout = 30 * time.Second
)

// Client calls the Google Ads Keyword Planner API.
type Client struct {
	httpClient      *http.Client
	developerToken  string
	customerID      string
	loginCustomerID string
	baseURL         string
	tokenSource     oauth2.TokenSource
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
func (c *Client) GenerateKeywordIdeas(
	ctx context.Context,
	seedKeywords []string,
	seedURL string,
	language string,
) (*KeywordIdeasResponse, error) {
	reqBody := c.buildKeywordIdeasRequest(seedKeywords, seedURL, language)
	endpoint := fmt.Sprintf("%s/customers/%s:generateKeywordIdeas", c.baseURL, c.customerID)

	var raw generateKeywordIdeasResponse
	if err := c.post(ctx, endpoint, reqBody, &raw); err != nil {
		return nil, err
	}

	ideas := make([]KeywordIdea, 0, len(raw.Results))
	for _, r := range raw.Results {
		ideas = append(ideas, KeywordIdea{
			Text:                   r.Text,
			AvgMonthlySearches:     parseI64(r.KeywordIdeaMetrics.AvgMonthlySearches),
			Competition:            r.KeywordIdeaMetrics.Competition,
			LowTopOfPageBidMicros:  parseI64(r.KeywordIdeaMetrics.LowTopOfPageBidMicros),
			HighTopOfPageBidMicros: parseI64(r.KeywordIdeaMetrics.HighTopOfPageBidMicros),
		})
	}

	return &KeywordIdeasResponse{
		SeedKeywords: seedKeywords,
		URL:          seedURL,
		Ideas:        ideas,
		Count:        len(ideas),
	}, nil
}

// GetHistoricalMetrics returns historical search metrics for a list of keywords.
func (c *Client) GetHistoricalMetrics(
	ctx context.Context,
	keywords []string,
) (*HistoricalMetricsResponse, error) {
	reqBody := generateHistoricalMetricsRequest{Keywords: keywords}
	endpoint := fmt.Sprintf("%s/customers/%s:generateKeywordHistoricalMetrics", c.baseURL, c.customerID)

	var raw generateHistoricalMetricsResponse
	if err := c.post(ctx, endpoint, reqBody, &raw); err != nil {
		return nil, err
	}

	metrics := make([]KeywordMetrics, 0, len(raw.Metrics))
	for _, r := range raw.Metrics {
		monthly := make([]MonthlyVolume, 0, len(r.KeywordMetrics.MonthlySearchVolumes))
		for _, m := range r.KeywordMetrics.MonthlySearchVolumes {
			monthly = append(monthly, MonthlyVolume{
				Year:            m.Year,
				Month:           parseMonthEnum(m.Month),
				MonthlySearches: parseI64(m.MonthlySearches),
			})
		}
		metrics = append(metrics, KeywordMetrics{
			Text:                   r.Text,
			AvgMonthlySearches:     parseI64(r.KeywordMetrics.AvgMonthlySearches),
			Competition:            r.KeywordMetrics.Competition,
			CompetitionIndex:       r.KeywordMetrics.CompetitionIndex,
			LowTopOfPageBidMicros:  parseI64(r.KeywordMetrics.LowTopOfPageBidMicros),
			HighTopOfPageBidMicros: parseI64(r.KeywordMetrics.HighTopOfPageBidMicros),
			MonthlySearchVolumes:   monthly,
		})
	}

	return &HistoricalMetricsResponse{Keywords: metrics, Count: len(metrics)}, nil
}

// GetKeywordForecast returns projected performance metrics for a set of keywords.
func (c *Client) GetKeywordForecast(
	ctx context.Context,
	keywords []string,
	maxCPCMicros int64,
	forecastDays int,
) (*ForecastResponse, error) {
	if forecastDays <= 0 {
		forecastDays = 30
	}
	now := time.Now().UTC()
	startDate := now.Format("2006-01-02")
	endDate := now.AddDate(0, 0, forecastDays).Format("2006-01-02")

	biddable := make([]adGroupForecastKeyword, 0, len(keywords))
	for _, kw := range keywords {
		biddable = append(biddable, adGroupForecastKeyword{
			Keyword: forecastKeyword{Text: kw, MatchType: "BROAD"},
		})
	}

	reqBody := generateForecastMetricsRequest{
		CampaignForecastSpec: campaignForecastSpec{
			BiddingStrategy: biddingStrategy{
				ManualCpcBiddingStrategy: manualCpcBiddingStrategy{
					MaxCPCBidMicros: strconv.FormatInt(maxCPCMicros, 10),
				},
			},
			StartDate: startDate,
			EndDate:   endDate,
			AdGroups:  []adGroupForecast{{Biddable: biddable}},
		},
	}

	endpoint := fmt.Sprintf("%s/customers/%s:generateKeywordForecastMetrics", c.baseURL, c.customerID)

	var raw generateForecastMetricsResponse
	if err := c.post(ctx, endpoint, reqBody, &raw); err != nil {
		return nil, err
	}

	var forecastMetrics []KeywordForecastMetrics
	for _, ag := range raw.AdGroupForecastMetrics {
		for _, kf := range ag.KeywordForecastMetrics {
			forecastMetrics = append(forecastMetrics, KeywordForecastMetrics{
				Text:        kf.Keyword.Text,
				Impressions: kf.Metrics.Impressions,
				Clicks:      kf.Metrics.Clicks,
				CostMicros:  kf.Metrics.CostMicros,
				CTR:         kf.Metrics.CTR,
			})
		}
	}

	return &ForecastResponse{
		Keywords:     forecastMetrics,
		ForecastDays: forecastDays,
		MaxCPCMicros: maxCPCMicros,
	}, nil
}

func (c *Client) post(ctx context.Context, endpoint string, body, out any) error {
	reqBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshalling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBytes))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("developer-token", c.developerToken)
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

func (c *Client) buildKeywordIdeasRequest(seedKeywords []string, seedURL, language string) generateKeywordIdeasRequest {
	req := generateKeywordIdeasRequest{Language: language}
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


