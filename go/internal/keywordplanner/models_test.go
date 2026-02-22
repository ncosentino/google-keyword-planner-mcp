package keywordplanner_test

import (
	"testing"

	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

func TestNewClient_NotNil(t *testing.T) {
	t.Parallel()
	client := keywordplanner.NewClient("dev-token", "client-id", "client-secret", "refresh-token", "1234567890")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestKeywordIdeasResponse_Count(t *testing.T) {
	t.Parallel()
	resp := &keywordplanner.KeywordIdeasResponse{
		Ideas: []keywordplanner.KeywordIdea{
			{Text: "golang tutorial", AvgMonthlySearches: 5000, Competition: "LOW"},
			{Text: "go programming", AvgMonthlySearches: 8000, Competition: "MEDIUM"},
		},
		Count: 2,
	}
	if resp.Count != len(resp.Ideas) {
		t.Errorf("Count = %d, want %d", resp.Count, len(resp.Ideas))
	}
}

func TestHistoricalMetricsResponse_Count(t *testing.T) {
	t.Parallel()
	resp := &keywordplanner.HistoricalMetricsResponse{
		Keywords: []keywordplanner.KeywordMetrics{
			{Text: "blazor", AvgMonthlySearches: 12000, Competition: "LOW"},
		},
		Count: 1,
	}
	if resp.Count != len(resp.Keywords) {
		t.Errorf("Count = %d, want %d", resp.Count, len(resp.Keywords))
	}
}
