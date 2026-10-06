package server_test

import (
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person installing a model server does not have to come back and press
// Check again: the card carries what it says in a few words, the page asks
// for them again (31-connect-wait.js), and they change when what is on this
// computer does, so the page follows.
func TestTheConnectCardNoticesWhatIsInstalled(t *testing.T) {
	was := llm.DefaultCandidates
	llm.DefaultCandidates = nil
	defer func() { llm.DefaultCandidates = was }()

	_, h := newApp(t)
	page := get(t, h, "/chat").Body.String()
	m := regexp.MustCompile(`class="sw-connect[^"]*" data-wait="([^"]*)"`).FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("the card carries what it says; body: %s", truncate(page))
	}
	if now := get(t, h, "/model/wait").Body.String(); now != html.UnescapeString(m[1]) {
		t.Errorf("asked again with nothing changed, the words are the same: %q, then %q", html.UnescapeString(m[1]), now)
	}

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			io.WriteString(w, `{"data":[{"id":"llama3.1"}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer ollama.Close()
	llm.DefaultCandidates = []llm.Candidate{{Server: "Ollama", BaseURL: ollama.URL + "/v1"}}
	postForm(t, h, "/model/check", url.Values{"from": {"/chat"}})
	if now := get(t, h, "/model/wait").Body.String(); now == html.UnescapeString(m[1]) {
		t.Errorf("a model server installed changes the words, so the page follows: still %q", now)
	}

	js, err := os.ReadFile("../../design/base/31-connect-wait.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`fetch("/model/wait"`, `getAttribute("data-wait")`, "window.swRefresh(0)", "typing(card)"} {
		if !strings.Contains(string(js), want) {
			t.Errorf("31-connect-wait.js does not have %s", want)
		}
	}
}
