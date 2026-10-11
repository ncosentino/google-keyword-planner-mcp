package keywordplanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

func TestKeywordIdeasRanksAllPagesBeforeLimiting(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			_, _ = w.Write([]byte(`{"nextPageToken":"second","results":[{"text":"small","keywordIdeaMetrics":{"avgMonthlySearches":"10"}}]}`))
			return
		}
		if body["pageToken"] != "second" || body["language"] != "languageConstants/1000" {
			t.Errorf("next-page request did not preserve parameters: %v", body)
		}
		_, _ = w.Write([]byte(`{"results":[{"text":"large","keywordIdeaMetrics":{"avgMonthlySearches":"900"}}]}`))
	}))
	defer server.Close()
	client := keywordplanner.NewTestClient("token", "123", "", server.URL, server.Client())
	result, err := client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "languageConstants/1000", nil, "", keywordplanner.IdeaOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || result.TotalFound != 2 || result.Count != 1 || result.Ideas[0].Text != "large" {
		t.Fatalf("requests = %d, result = %+v", requests, result)
	}
}

func TestBlankTargetingDoesNotSilentlyOverrideDefaults(t *testing.T) {
	client := keywordplanner.NewTestClient("token", "123", "", "http://unused.invalid", &http.Client{})
	client.SetDefaultTargeting("languageConstants/1000", []string{"geoTargetConstants/2840"})
	_, err := client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", " ", nil, "", keywordplanner.IdeaOptions{})
	if err == nil {
		t.Fatal("blank language accepted")
	}
	_, err = client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", []string{" "}, "", keywordplanner.IdeaOptions{})
	if err == nil {
		t.Fatal("blank geography accepted")
	}
}
