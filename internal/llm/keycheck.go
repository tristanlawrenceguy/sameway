package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// A key pasted with a character missing, or one taken back on the
// provider's website, was kept without a word, and the person learned of it
// from their first message failing. A key is now asked about before it is
// kept: the provider says whether it knows it, and OpenRouter whether it
// has credit left. Asking costs nothing and sends no conversation.

// KeyCheck is where keys are asked about. The zero value is the
// providers' own addresses; an app is given its own through app.Options,
// so a test points one at a fake while others run beside it.
type KeyCheck struct {
	Anthropic, OpenRouter string
}

// at is where a key in env is asked about, or "" when it cannot be.
func (k KeyCheck) at(env string) string {
	switch {
	case env == "ANTHROPIC_API_KEY" && k.Anthropic != "":
		return k.Anthropic
	case env == "ANTHROPIC_API_KEY":
		return "https://api.anthropic.com/v1/models?limit=1"
	case env == "OPENROUTER_API_KEY" && k.OpenRouter != "":
		return k.OpenRouter
	case env == "OPENROUTER_API_KEY":
		return "https://openrouter.ai/api/v1/key"
	}
	return ""
}

// KeyVerdict is what a provider said of a key.
type KeyVerdict int

const (
	// KeyGood is a key the provider knows.
	KeyGood KeyVerdict = iota
	// KeyRefused is a key the provider does not know or has taken back.
	KeyRefused
	// KeyNoCredit is a key it knows with nothing left to spend.
	KeyNoCredit
	// KeyUnchecked is a key that could not be asked about: no connection,
	// or the provider answering something else.
	KeyUnchecked
)

// Check asks the provider the key's environment variable names whether it
// knows the key.
func (k KeyCheck) Check(ctx context.Context, env, key string) (KeyVerdict, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var req *http.Request
	var err error
	switch env {
	case "ANTHROPIC_API_KEY":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, k.at(env), nil)
		if err == nil {
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
		}
	case "OPENROUTER_API_KEY":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, k.at(env), nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+key)
		}
	default:
		return KeyUnchecked, fmt.Errorf("no way to check a key for %s", env)
	}
	if err != nil {
		return KeyUnchecked, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return KeyUnchecked, err
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return KeyRefused, nil
	case res.StatusCode != http.StatusOK:
		return KeyUnchecked, fmt.Errorf("the provider answered %s", res.Status)
	}
	if env == "OPENROUTER_API_KEY" {
		var body struct {
			Data struct {
				LimitRemaining *float64 `json:"limit_remaining"`
			} `json:"data"`
		}
		if json.NewDecoder(res.Body).Decode(&body) == nil && body.Data.LimitRemaining != nil && *body.Data.LimitRemaining <= 0 {
			return KeyNoCredit, nil
		}
	}
	return KeyGood, nil
}

// A saved key is asked about again now and then, so one taken back or out
// of credit is said where the person can paste another, not found out from
// a message that fails. Asked at most every ten minutes for each key, at
// each address it is asked at.
var keySeen sync.Map // address and key -> keyAsked

type keyAsked struct {
	at      time.Time
	verdict KeyVerdict
}

// keyStillGood is Answers for a saved key: refused or out of credit is not
// answering, said by company; a key that could not be asked about is taken
// to be good. A key the provider cannot be asked about is never asked.
func keyStillGood(ctx context.Context, cfg Config, company string) (bool, string) {
	env := cfg.APIKeyEnv
	key, at := cfg.Keys.Get(env), cfg.KeyCheck.at(env)
	if at == "" || key == "" {
		return true, ""
	}
	v, ok := keySeen.Load(at + " " + key)
	if !ok || time.Since(v.(keyAsked).at) > 10*time.Minute {
		verdict, _ := cfg.KeyCheck.Check(ctx, env, key)
		v = keyAsked{time.Now(), verdict}
		keySeen.Store(at+" "+key, v)
	}
	switch v.(keyAsked).verdict {
	case KeyRefused:
		return false, company + " no longer accepts the key saved on this computer. Paste a new one."
	case KeyNoCredit:
		return false, "The " + company + " key saved on this computer has no credit left. Add some on their website, or paste another key."
	}
	return true, ""
}
