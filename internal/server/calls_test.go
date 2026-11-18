package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/twilio"
)

// fakeTwilio keeps the text messages sent.
type fakeTwilio struct {
	mu    sync.Mutex
	texts []url.Values
}

func (f *fakeTwilio) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	f.mu.Lock()
	f.texts = append(f.texts, r.PostForm)
	f.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"sid":"SM1"}`))
}

// A phone call is one more way in: only Twilio's signed requests are
// taken; the caller is greeted, heard and answered aloud as a visitor to
// the published pages, the answer worked out while they hear one moment;
// what they ask for waits for the owner, whose answer goes back to them by
// text message from the number they rang.
func TestAPhoneCallIsOneMoreWayIn(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	internet := h.(*server.Server).Public(nil)
	sms := &fakeTwilio{}
	a.Automate.HTTP = &http.Client{Transport: served{sms}}
	postForm(t, h, "/calls/connect", url.Values{"sid": {"AC1"}, "token": {"tok"}})
	call := func(path string, form url.Values, sign bool) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if sign {
			req.Header.Set("X-Twilio-Signature", twilio.Account{Token: "tok"}.Sign("https://example.com"+path, form))
		}
		rec := httptest.NewRecorder()
		internet.ServeHTTP(rec, req)
		if sign && rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}
	base := url.Values{"CallSid": {"CA1"}, "From": {"+447700900001"}, "To": {"+442071234567"}}
	if req := httptest.NewRequest(http.MethodPost, "/calls/voice", strings.NewReader(base.Encode())); true {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		internet.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("a request without Twilio's signature is not a call: %d", rec.Code)
		}
	}
	if got := call("/calls/voice", base, true); !strings.Contains(got, "<Gather input=\"speech\"") || !strings.Contains(got, "How can I help?") {
		t.Fatalf("the caller is greeted and heard: %s", got)
	}
	a.Chat.Provider = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Visit for Pat", "starts": "2026-03-17T15:00:00Z"}}),
		{Text: "I have asked for Tuesday at three. You will get a text when it is answered."},
	}}
	heard := url.Values{"CallSid": {"CA1"}, "From": {"+447700900001"}, "To": {"+442071234567"}, "SpeechResult": {"Could someone come on Tuesday at 3pm? I'm Pat."}}
	if got := call("/calls/voice", heard, true); !strings.Contains(got, "One moment") {
		t.Fatalf("they hear one moment while the answer is worked out: %s", got)
	}
	var said string
	for deadline := time.Now().Add(5 * time.Second); !strings.Contains(said, "Tuesday at three") && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		said = call("/calls/voice?wait=1", base, true)
	}
	if !strings.Contains(said, "<Say>I have asked for Tuesday at three") || !strings.Contains(said, "<Gather") {
		t.Fatalf("and then the answer, aloud, listening again: %s", said)
	}
	asks := a.Chat.Suggestions()
	if len(asks) != 1 {
		t.Fatalf("what they asked for waits for the owner: %d", len(asks))
	}
	wantStatus(t, postForm(t, h, "/suggested/yes", url.Values{"id": {asks[0].ID}}), http.StatusSeeOther)
	sms.mu.Lock()
	defer sms.mu.Unlock()
	if len(sms.texts) != 1 || sms.texts[0].Get("To") != "+447700900001" || sms.texts[0].Get("From") != "+442071234567" || !strings.Contains(sms.texts[0].Get("Body"), "said yes") {
		t.Errorf("the answer goes back by text, from the number they rang: %v", sms.texts)
	}
}

// held is a model that answers only when let go, so a call can be heard
// while its answer is still being worked out.
type held struct{ open chan struct{} }

func (h held) Name() string { return "held" }

func (h held) Complete(ctx context.Context, _ llm.Request) (*llm.Response, error) {
	<-h.open
	return &llm.Response{Text: "We are open weekdays."}, nil
}

// A call works for whoever rings: they have time to start and may pause
// mid-sentence; saying nothing hears it again before goodbye; and a long
// wait for an answer is not silence.
func TestAPhoneCallGivesTheCallerTime(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	internet := h.(*server.Server).Public(nil)
	a.Automate.HTTP = &http.Client{Transport: served{&fakeTwilio{}}}
	postForm(t, h, "/calls/connect", url.Values{"sid": {"AC1"}, "token": {"tok"}})
	call := func(path string, form url.Values) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Twilio-Signature", twilio.Account{Token: "tok"}.Sign("https://example.com"+path, form))
		rec := httptest.NewRecorder()
		internet.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	base := url.Values{"CallSid": {"CA2"}, "From": {"+447700900002"}, "To": {"+442071234567"}}
	hello := call("/calls/voice", base)
	if strings.Contains(hello, `speechTimeout="auto"`) || !strings.Contains(hello, `timeout="8" speechTimeout="3"`) {
		t.Errorf("the caller has time to start, and to pause: %s", hello)
	}
	if strings.Count(hello, "<Gather") != 2 || !strings.Contains(hello, "I did not hear anything. How can I help?</Say></Gather><Say>Goodbye.") {
		t.Errorf("saying nothing hears it again before goodbye: %s", hello)
	}
	release := make(chan struct{})
	a.Chat.Provider = held{release}
	heard := url.Values{"CallSid": {"CA2"}, "From": {"+447700900002"}, "To": {"+442071234567"}, "SpeechResult": {"When are you open?"}}
	call("/calls/voice", heard)
	waited := ""
	for range 4 {
		waited += call("/calls/voice?wait=1", base)
	}
	if !strings.Contains(waited, "Still working on it.") {
		t.Errorf("a long wait is not silence: %s", waited)
	}
	close(release)
	var said string
	for deadline := time.Now().Add(5 * time.Second); !strings.Contains(said, "weekdays") && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		said = call("/calls/voice?wait=1", base)
	}
	if !strings.Contains(said, "<Say>Here it is again. We are open weekdays.</Say>") {
		t.Errorf("the answer is said again to a caller who says nothing: %s", said)
	}
}
