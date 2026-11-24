// Package twilio checks that a call's request came from Twilio, sends a
// text message from the owner's Twilio number and rings a number from it,
// with their account's own ID and token. It is Twilio's API over https,
// and nothing else.
package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	form := url.Values{"From": {from}, "To": {to}, "Body": {body}}
	return a.call(ctx, http.MethodPost, "/Messages.json", strings.NewReader(form.Encode()), nil)
}

// Number is the account's first phone number, the one a call rings from.
func (a Account) Number(ctx context.Context) (string, error) {
	var got struct {
		Numbers []struct {
			Phone string `json:"phone_number"`
		} `json:"incoming_phone_numbers"`
	}
	if err := a.call(ctx, http.MethodGet, "/IncomingPhoneNumbers.json?PageSize=1", nil, &got); err != nil {
		return "", err
	}
	if len(got.Numbers) == 0 || got.Numbers[0].Phone == "" {
		return "", errors.New("the Twilio account has no phone number to ring from: get one in its console")
	}
	return got.Numbers[0].Phone, nil
}

// Dial rings to from one of the account's numbers. Once it is picked up
// Twilio asks answer what to say, and when it ends it tells ended how.
func (a Account) Dial(ctx context.Context, from, to, answer, ended string) error {
	form := url.Values{"From": {from}, "To": {to}, "Url": {answer}, "Method": {"POST"},
		"StatusCallback": {ended}, "StatusCallbackMethod": {"POST"}, "Timeout": {"30"}}
	return a.call(ctx, http.MethodPost, "/Calls.json", strings.NewReader(form.Encode()), nil)
}

// Check says whether Twilio takes the account's ID and token.
func (a Account) Check(ctx context.Context) error {
	err := a.call(ctx, http.MethodGet, ".json", nil, nil)
	if errors.Is(err, errRefused) {
		return errors.New("Twilio did not take that Account SID and Auth token: copy both again from its console")
	}
	return err
}

var errRefused = errors.New("Twilio did not take the account's ID and token")

// call asks Twilio about the account, at path after its address, and
// reads its answer into out, unless out is nil.
func (a Account) call(ctx context.Context, method, path string, body io.Reader, out any) error {
	base, client := a.Base, a.HTTP
	if base == "" {
		base = API
	}
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, method, base+"/2010-04-01/Accounts/"+url.PathEscape(a.SID)+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.SID, a.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, err := client.Do(req)
	if err != nil {
		return errors.New("Twilio could not be reached")
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusNotFound:
		return errRefused
	case res.StatusCode >= 300:
		return fmt.Errorf("Twilio answered %s", res.Status)
	case out != nil:
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}
