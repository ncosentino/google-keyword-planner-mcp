package keywordplanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

// forecastKeywords returns the keyword texts of a forecast request body.
func forecastKeywords(t *testing.T, r *http.Request) []string {
	t.Helper()
	var body struct {
		Campaign struct {
			AdGroups []struct {
				Keywords []struct {
					Text string `json:"text"`
				} `json:"keywords"`
			} `json:"adGroups"`
		} `json:"campaign"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Campaign.AdGroups) != 1 {
		t.Errorf("unexpected forecast request body: %v", err)
		return nil
	}
	texts := make([]string, 0, len(body.Campaign.AdGroups[0].Keywords))
	for _, kw := range body.Campaign.AdGroups[0].Keywords {
		texts = append(texts, kw.Text)
	}
	return texts
}

func TestGetKeywordForecast_SingleKeywordNeedsOneRequest(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"campaignForecastMetrics": {"clicks": 7.5, "costMicros": "3000000"}}`))
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	result, err := client.GetKeywordForecast(context.Background(), keywordplanner.ForecastRequest{Keywords: []string{"go"}, IncludeKeywordBreakdown: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := requests.Load(); got != 1 {
		t.Errorf("made %d requests, want 1", got)
	}
	if result.Keywords[0].Text != "go" || result.Keywords[0].Clicks == nil || *result.Keywords[0].Clicks != 7.5 {
		t.Errorf("Keywords[0] = %+v, want go with 7.5 clicks", result.Keywords[0])
	}
	if result.Total.Clicks == nil || *result.Total.Clicks != 7.5 || result.Total.CostMicros == nil || *result.Total.CostMicros != 3000000 || result.Total.Text != "" {
		t.Errorf("Total = %+v, want the single keyword's metrics without a text", result.Total)
	}
}

func TestGetKeywordForecast_RunsRequestsConcurrentlyAndKeepsOrder(t *testing.T) {
	t.Parallel()

	keywords := []string{"a", "b", "c", "d"}
	clicks := map[string]float64{"a": 1, "b": 2, "c": 3, "d": 4}

	var inFlight, maxInFlight atomic.Int32
	entered := make(chan struct{}, len(keywords))
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			seen := maxInFlight.Load()
			if now <= seen || maxInFlight.CompareAndSwap(seen, now) {
				break
			}
		}
		texts := forecastKeywords(t, r)
		value := 100.0
		if len(texts) == 1 {
			value = clicks[texts[0]]
			entered <- struct{}{}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"campaignForecastMetrics": map[string]any{"clicks": value}})
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		for range keywords {
			select {
			case <-entered:
			case <-ctx.Done():
				return
			}
		}
		close(release)
	}()
	result, err := client.GetKeywordForecast(ctx, keywordplanner.ForecastRequest{Keywords: keywords, IncludeKeywordBreakdown: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := maxInFlight.Load(); got < 2 || got > 5 {
		t.Errorf("max concurrent requests = %d, want between 2 and 5", got)
	}
	for i, kw := range keywords {
		if result.Keywords[i].Text != kw || result.Keywords[i].Clicks == nil || *result.Keywords[i].Clicks != clicks[kw] {
			t.Errorf("Keywords[%d] = %+v, want %s with %v clicks", i, result.Keywords[i], kw, clicks[kw])
		}
	}
	if result.Total.Clicks == nil || *result.Total.Clicks != 100 {
		t.Errorf("Total.Clicks = %v, want 100", result.Total.Clicks)
	}
}

func TestGetKeywordForecast_ReportsTheFailingKeyword(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		texts := forecastKeywords(t, r)
		w.Header().Set("Content-Type", "application/json")
		if len(texts) == 1 && texts[0] == "bad" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": {"message": "nope"}}`))
			return
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := keywordplanner.NewTestClient("dev-token", "123", "", srv.URL, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := client.GetKeywordForecast(ctx, keywordplanner.ForecastRequest{
		Keywords:                []string{"good", "bad", "fine"},
		IncludeKeywordBreakdown: true,
	})
	if err == nil || !strings.Contains(err.Error(), `forecasting "bad"`) {
		t.Fatalf("err = %v, want the failure of keyword \"bad\"", err)
	}
	if result != nil {
		t.Errorf("result = %+v, want nil on failure", result)
	}
}
