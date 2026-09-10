package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

func TestDetectFindsServersWithModels(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"object":"list","data":[{"id":"qwen3"},{"id":"llama3.1"}]}`))
	}))
	defer good.Close()
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	}))
	defer empty.Close()
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>not an api</html>`))
	}))
	defer broken.Close()

	found := llm.Detect(context.Background(), []llm.Candidate{
		{"dead", "http://127.0.0.1:1/v1"},
		{"empty", empty.URL + "/v1"},
		{"broken", broken.URL + "/v1"},
		{"good", good.URL + "/v1"},
	})
	if len(found) != 1 || found[0].Server != "good" || found[0].Model != "qwen3" || len(found[0].Models) != 2 {
		t.Fatalf("detect: %+v", found)
	}
	if found[0].BaseURL != good.URL+"/v1" {
		t.Errorf("base url: %s", found[0].BaseURL)
	}
}

func TestDetectWithNothingRunning(t *testing.T) {
	if found := llm.Detect(context.Background(), []llm.Candidate{{"dead", "http://127.0.0.1:1/v1"}}); len(found) != 0 {
		t.Errorf("expected nothing, got %+v", found)
	}
}
