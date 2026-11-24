package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/twilio"
)

// Phoning a business with one question: nothing is rung before the
// owner's yes, put to them with the number, who, the exact question and
// what the call will and will not do; the call says it is an AI and asks
// only that, at an address only Twilio's signature opens; what they
// answer is told to the owner, in their chat and as a notification, and
// so is a call nobody answered.
func TestPhoneABusinessWithOneQuestion(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	internet := h.(*server.Server).Public(nil)
	tw := &fakeTwilio{}
	a.Automate.HTTP = &http.Client{Transport: served{tw}}
	a.Chat.Address = func() string { return "https://example.com" }
	rang := make(chan string, 4)
	h.(*server.Server).OnRing(func(title, text, _ string) { rang <- title + ": " + text })
	ask := `{"number":"+441632960001","who":"Corner Bakery","question":"Are you open on Sunday?"}`
	if text, isErr := a.Chat.Call("phone_to_ask", json.RawMessage(ask)); !isErr || !strings.Contains(text, "unknown tool") && !strings.Contains(text, "not connected") {
		t.Errorf("without Twilio there is no phoning: %s", text)
	}
	postForm(t, h, "/calls/connect", url.Values{"sid": {"AC1"}, "token": {"tok"}})
	if text, isErr := a.Chat.Call("phone_to_ask", json.RawMessage(`{"number":"01632 960001","who":"Corner Bakery","question":"Open Sunday?"}`)); !isErr || !strings.Contains(text, "international form") || len(a.Chat.Proposals()) != 0 {
		t.Errorf("a number not in international form is refused, not asked: %s", text)
	}
	a.Chat.Call("phone_to_ask", json.RawMessage(ask))
	ps := a.Chat.Proposals()
	if len(ps) != 1 || len(tw.calls) != 0 {
		t.Fatalf("nothing is rung before the owner's yes: %+v %v", ps, tw.calls)
	}
	detail, _ := ps[0].Fields["detail"].(string)
	for _, want := range []string{`"Are you open on Sunday?"`, "AI assistant calling for you", "ask only this", "saves what they say", "books, pays, shares and agrees to nothing", "can't be taken back"} {
		if !strings.Contains(detail, want) {
			t.Errorf("the card says %q: %s", want, detail)
		}
	}
	if ps[0].Fields["summary"] != "Phone Corner Bakery on +441632960001?" || ps[0].Fields["yes"] != "Phone +441632960001" {
		t.Errorf("the card and its button name the number: %v / %v", ps[0].Fields["summary"], ps[0].Fields["yes"])
	}
	wantStatus(t, postForm(t, h, "/proposal/"+ps[0].ID+"/accept", nil), http.StatusSeeOther)
	tw.mu.Lock()
	if len(tw.calls) != 1 || tw.calls[0].Get("To") != "+441632960001" || tw.calls[0].Get("From") != "+442071234567" {
		t.Fatalf("the yes rings the number from the owner's: %v", tw.calls)
	}
	answer := strings.TrimPrefix(tw.calls[0].Get("Url"), "https://example.com")
	tw.mu.Unlock()

	step := func(path string, form url.Values, signature string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Twilio-Signature", signature)
		rec := httptest.NewRecorder()
		internet.ServeHTTP(rec, req)
		return rec
	}
	signed := func(path string, form url.Values) string {
		return twilio.Account{Token: "tok"}.Sign("https://example.com"+path, form)
	}
	picked := url.Values{"CallSid": {"CA9"}, "CallStatus": {"in-progress"}}
	if rec := step(answer, picked, twilio.Account{Token: "wrong"}.Sign("https://example.com"+answer, picked)); rec.Code != http.StatusForbidden || strings.Contains(rec.Body.String(), "Sunday") {
		t.Errorf("a request without Twilio's signature is refused: %d %s", rec.Code, rec.Body.String())
	}
	script := step(answer, picked, signed(answer, picked)).Body.String()
	if !strings.Contains(script, "<Say>Hello, this is an AI assistant calling for "+a.Workspace.Config.Name+", with one question. This call is transcribed.</Say>") ||
		strings.Count(script, "<Say>Are you open on Sunday?</Say></Gather>") != 2 || !strings.HasSuffix(script, "<Say>Thank you. Goodbye.</Say><Hangup/></Response>") {
		t.Fatalf("the call says it is an AI, asks the question, once more if unheard, and thanks them: %s", script)
	}
	heard := url.Values{"CallSid": {"CA9"}, "CallStatus": {"in-progress"}, "SpeechResult": {"Yes, ten till four."}}
	if got := step(answer, heard, signed(answer, heard)).Body.String(); !strings.Contains(got, "Thank you. Goodbye.</Say><Hangup/>") || strings.Contains(got, "<Gather") {
		t.Errorf("an answer is thanked and the call ends: %s", got)
	}
	ended := url.Values{"CallSid": {"CA9"}, "CallStatus": {"completed"}}
	step(answer+"?ended=1", ended, signed(answer+"?ended=1", ended))
	want := `Corner Bakery answered "Are you open on Sunday?": "Yes, ten till four.".`
	if got := waitRing(t, rang); !strings.Contains(got, want) {
		t.Errorf("the owner is told: %s", got)
	}
	msgs, _ := a.Chat.Messages()
	if len(msgs) == 0 || msgs[len(msgs)-1].Fields["content"] != want {
		t.Errorf("in their chat too: %d messages", len(msgs))
	}
	if len(rang) != 0 {
		t.Errorf("once: %s", <-rang)
	}

	a.Chat.Call("phone_to_ask", json.RawMessage(`{"number":"+441632960002","who":"Hill Garage","question":"Is my car ready?"}`))
	ps = a.Chat.Proposals()
	wantStatus(t, postForm(t, h, "/proposal/"+ps[len(ps)-1].ID+"/accept", nil), http.StatusSeeOther)
	tw.mu.Lock()
	ended2 := strings.TrimPrefix(tw.calls[len(tw.calls)-1].Get("StatusCallback"), "https://example.com")
	tw.mu.Unlock()
	none := url.Values{"CallSid": {"CA10"}, "CallStatus": {"no-answer"}}
	step(ended2, none, signed(ended2, none))
	if got := waitRing(t, rang); !strings.Contains(got, `Hill Garage did not answer the phone, so "Is my car ready?" was not asked.`) {
		t.Errorf("a call nobody answered is told too: %s", got)
	}
}

func waitRing(t *testing.T, rang chan string) string {
	t.Helper()
	select {
	case got := <-rang:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("the owner was not told")
		return ""
	}
}
