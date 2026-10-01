package server_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// blinkered is a model that cannot see, under a name of its own: what a
// model has shown of its sight is remembered by name.
type blinkered struct{ sees }

func (*blinkered) Name() string { return "blinkered" }

// The help page says what this computer and its model cannot do before a
// person finds out by trying: pictures, once the model has been sent one,
// and speech-to-text, with the owner's press to get it.
func TestTheHelpPageSaysWhatThisComputerCannotDo(t *testing.T) {
	a, h := newApp(t)
	model := &blinkered{sees{blind: true}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	h.(*server.Server).UseSpeech(server.Speech{Ready: func() bool { return false }})

	page := get(t, h, "/help").Body.String()
	if !strings.Contains(page, "What this computer can do") || !strings.Contains(page, "blinkered, has not been sent one yet") {
		t.Errorf("before a picture, the page says it is not known: %.2000s", page)
	}
	if !strings.Contains(page, "not on this computer yet") || !strings.Contains(page, `action="/speech/get"`) {
		t.Errorf("no speech-to-text is said, with the owner's press: %.2000s", page)
	}

	body, ct := multipartFile(t, "Fern.png", string(pngOf(t, 40, 30)), nil)
	id := strings.TrimPrefix(do(t, h, "POST", "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	if _, err := a.Chat.SendFile(context.Background(), "", "What is this?", id); err != nil {
		t.Fatal(err)
	}
	if page := get(t, h, "/help").Body.String(); !strings.Contains(page, "blinkered, cannot see them") {
		t.Errorf("after a picture it could not see, the page says so: %.2000s", page)
	}
}
