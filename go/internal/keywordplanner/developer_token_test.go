package keywordplanner_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ncosentino/google-keyword-planner-mcp/go/internal/keywordplanner"
)

func TestKeywordIdeasOmitAbsentLegacyDeveloperToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, present := r.Header["Developer-Token"]; present {
			t.Error("absent legacy token was sent")
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	client := keywordplanner.NewTestClient("", "123", "", server.URL, server.Client())
	_, err := client.GenerateKeywordIdeas(context.Background(), []string{"go"}, "", "", nil, "", keywordplanner.IdeaOptions{})
	if err != nil {
		t.Fatal(err)
	}
}
