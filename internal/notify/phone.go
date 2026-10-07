package notify

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// A reminder rang on the computer, and a person away from it missed it.
// ntfy is a free notification service with an app for each phone: a ring
// posted to a topic reaches every phone subscribed to it. The topic is a
// long random name Sameway makes (internal/server phone.go), which is
// what keeps it the person's: whoever knows it can read what is sent. A link to this
// computer leads nowhere from a phone, so only a link elsewhere is sent.

// NtfyServer is where topics are made; tests point it elsewhere.
var NtfyServer = "https://ntfy.sh"

// toPhone posts a ring to an ntfy topic address.
func toPhone(topic, title, text, link string) error {
	req, err := http.NewRequest(http.MethodPost, topic, strings.NewReader(text))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Tags", "bell")
	if u, err := url.Parse(link); err == nil && u.Hostname() != "" && !local(u.Hostname()) {
		req.Header.Set("Click", link)
	}
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("%s answered %s", topic, res.Status)
	}
	return nil
}

func local(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// Phone sends one message to a topic, for the page's test.
func Phone(topic, title, text string) error { return toPhone(topic, title, text, "") }
