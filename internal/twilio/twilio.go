// Package twilio checks that a call's request came from Twilio, and sends
// a text message from the owner's Twilio number, with their account's own
// ID and token. It is Twilio's API over https, and nothing else.
package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// API is where Twilio answers.
const API = "https://api.twilio.com"

// Account is the owner's Twilio account.
type Account struct {
	SID, Token string
	HTTP       *http.Client // nil is one with a minute's limit
	Base       string       // "" is API; a test gives its own
}

// Signed says whether a request to address with form came from Twilio:
// its X-Twilio-Signature is the address and every form value, sorted by
// name, signed with the account's token (Twilio's request validation).
func (a Account) Signed(address string, form url.Values, signature string) bool {
	if a.Token == "" || signature == "" {
		return false
	}
	return hmac.Equal([]byte(a.Sign(address, form)), []byte(signature))
}

// Sign is the signature Twilio gives a request to address with form.
func (a Account) Sign(address string, form url.Values) string {
	names := make([]string, 0, len(form))
	for k := range form {
		names = append(names, k)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString(address)
	for _, k := range names {
		for _, v := range form[k] {
			b.WriteString(k + v)
		}
	}
	mac := hmac.New(sha1.New, []byte(a.Token))
	mac.Write([]byte(b.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Text sends a text message from one of the account's numbers.
func (a Account) Text(ctx context.Context, from, to, body string) error {
	base, client := a.Base, a.HTTP
	if base == "" {
		base = API
	}
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	form := url.Values{"From": {from}, "To": {to}, "Body": {body}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/2010-04-01/Accounts/"+url.PathEscape(a.SID)+"/Messages.json", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.SID, a.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return errors.New("Twilio could not be reached")
	}
	res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("Twilio answered %s", res.Status)
	}
	return nil
}
