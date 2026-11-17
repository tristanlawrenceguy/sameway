package server

import (
	"context"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/stripe"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Payments: the owner pastes a key from their own Stripe account, once,
// and the assistant can make links to pay them (chat/payments.go), each
// with their yes. A restricted key that may only write payment links and
// prices and read checkout sessions is enough, and is all it should be.

const stripeKey = "stripe:key"

func (s *Server) paymentsPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	if s.keys().Get(stripeKey) != "" {
		b.WriteString(`<p>Connected to your Stripe account. Ask the assistant for a link to pay you, for a quote, a deposit or a bill; each is put to you first, and you hear when one is paid.</p>`)
		b.WriteString(string(s.form(ui.Form{Action: "/payments/off", Button: &ui.Button{Label: "Disconnect Stripe", Variant: ui.Secondary}})))
		s.page(w, r, "Payments", template.HTML(b.String()), pageOptions{})
		return
	}
	b.WriteString(`<ol class="sw-stack"><li>In the <a class="sw-link" href="https://dashboard.stripe.com/apikeys">Stripe dashboard</a>, under API keys, create a restricted key.</li><li>Let it write Payment Links and Prices, and read Checkout Sessions; nothing else.</li><li>Paste it here.</li></ol>`)
	b.WriteString(string(s.form(ui.Form{Action: "/payments/connect", Class: "sw-stack",
		Body:   s.part(ui.TextField{Label: "Your Stripe key", Name: "key", Required: true, Hint: "It begins rk_. It stays on this computer, in your keys, not in the workspace."}),
		Button: &ui.Button{Label: "Connect"}})))
	s.page(w, r, "Payments", template.HTML(b.String()), pageOptions{Lede: "Links to pay you, through your own Stripe account."})
}

func (s *Server) paymentsConnect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := strings.TrimSpace(r.PostForm.Get("key"))
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	err := stripe.Account{Key: key, HTTP: s.app.Automate.HTTP}.Check(ctx)
	if err == nil {
		err = s.keys().Save(stripeKey, key)
	}
	if err != nil {
		s.failed(w, r, "Not connected", err, "/payments")
		return
	}
	s.tellAt(w, r, outcome{Title: "Stripe connected", Text: "Ask the assistant for a link to pay you."}, "/payments")
}

func (s *Server) paymentsOff(w http.ResponseWriter, r *http.Request) {
	s.keys().Save(stripeKey, "")
	s.tellAt(w, r, outcome{Title: "Stripe disconnected", Text: "Links already made still work; no new ones are made."}, "/payments")
}
