package keywordplanner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestForecastPeriodIsInclusiveAndFutureAcrossTimezones(t *testing.T) {
	for _, now := range []time.Time{
		time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC),
		time.Date(2028, 2, 28, 0, 0, 0, 0, time.UTC),
	} {
		for _, days := range []int{1, 30, 364} {
			period := forecastPeriodFor(now, days)
			start, err := time.Parse("2006-01-02", period.StartDate)
			if err != nil {
				t.Fatal(err)
			}
			end, err := time.Parse("2006-01-02", period.EndDate)
			if err != nil {
				t.Fatal(err)
			}
			if got := int(end.Sub(start).Hours()/24) + 1; got != days {
				t.Fatalf("inclusive window = %d, want %d", got, days)
			}
			for _, offset := range []int{-12, 14} {
				accountToday := now.In(time.FixedZone("account", offset*3600)).Format("2006-01-02")
				if period.StartDate <= accountToday {
					t.Fatalf("start %s is not after account today %s", period.StartDate, accountToday)
				}
			}
		}
	}
}

func TestTargetingDefaultsAndExplicitUnrestrictedOverrides(t *testing.T) {
	client := &Client{}
	client.SetDefaultTargeting("languageConstants/1000", []string{"geoTargetConstants/2840"})
	language, targets, err := client.targeting("", nil)
	if err != nil || language != "languageConstants/1000" || len(targets) != 1 {
		t.Fatalf("defaults = %s %v %v", language, targets, err)
	}
	language, targets, err = client.targeting("all", []string{})
	if err != nil || language != "" || len(targets) != 0 {
		t.Fatalf("unrestricted = %s %v %v", language, targets, err)
	}
	if _, _, err := client.targeting(" ", nil); err == nil {
		t.Fatal("blank language accepted")
	}
	if _, _, err := client.targeting("", []string{" "}); err == nil {
		t.Fatal("blank geography accepted")
	}
}

func TestForecastDefaultIsOneCombinedRequestAndPreservesMetricPresence(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"campaignForecastMetrics":{"clicks":0,"costMicros":"0"}}`))
	}))
	defer server.Close()
	client := NewTestClient("token", "123", "", server.URL, server.Client())
	result, err := client.GetKeywordForecast(context.Background(), ForecastRequest{Keywords: []string{"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 || len(result.Keywords) != 0 || result.Total.Clicks == nil || *result.Total.Clicks != 0 {
		t.Fatalf("requests = %d, result = %+v", requests, result)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, unavailable := range []string{"conversions", "averageCpaMicros", "averageCpcMicros", "impressions", "ctr"} {
		if strings.Contains(string(body), `"`+unavailable+`"`) {
			t.Fatalf("unavailable metric %s emitted in %s", unavailable, body)
		}
	}
}

type rejectingPlanningLimiter struct {
	calls int
}

func (limiter *rejectingPlanningLimiter) Wait(ctx context.Context) error {
	limiter.calls++
	return ctx.Err()
}

func TestPlanningQuotaCancellationPreventsHTTPRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("cancelled planning call reached HTTP server")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := NewTestClient("token", "123", "", server.URL, server.Client())
	limiter := &rejectingPlanningLimiter{}
	client.planningLimiter = limiter
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := client.post(ctx, server.URL+":generateKeywordForecastMetrics", struct{}{}, &struct{}{})
	if !errors.Is(err, context.Canceled) || limiter.calls != 1 {
		t.Fatalf("error = %v, quota calls = %d", err, limiter.calls)
	}

}

func TestForecastCancelledBeforeSchedulingReturnsNoResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := NewTestClient("token", "123", "", "http://unused.invalid", &http.Client{})
	result, err := client.GetKeywordForecast(ctx, ForecastRequest{Keywords: []string{"a", "b"}, IncludeKeywordBreakdown: true})
	if result != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
}
