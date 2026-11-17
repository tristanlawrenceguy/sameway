package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/stripe"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// Being paid: a link to pay the owner through their own Stripe account,
// made with their yes and kept as a payment, which the assistant puts in
// an email or a message as any link. Sameway reads Stripe now and then
// and marks a payment paid, logged, so an action can follow it: a receipt,
// a thank you. Nothing about a card or an account passes through Sameway;
// the money goes from whoever pays to the owner's Stripe.

// PaymentType is the kind a payment is kept as.
const PaymentType = "payment"

var paymentLinkOp = Op{Title: "Make a link to pay you",
	Access: ForOwner, Traits: Traits{OpenWorld: true},
	Words: []string{"pay", "payment", "invoice", "bill", "charge", "deposit", "stripe"},
	Doing: saying("Making a link to pay you"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		p, err := s.paymentOf(call.Args)
		if err != nil {
			return fail("%v", err), true
		}
		return s.ask(question{fmt.Sprintf("Make a link to pay you %s for %q?", money(p.Amount, p.Currency), p.What),
			fmt.Sprintf("Stripe would make a link that takes one payment of %s into your Stripe account, then closes. Check the amount: it is what they would pay.", money(p.Amount, p.Currency)),
			"Make a " + money(p.Amount, p.Currency) + " link", "Don't"},
			map[string]any{"tool": "payment_link", "what": p.What, "amount": p.Amount, "currency": p.Currency}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.paymentLink(call.Args) },
	Offered: func(s *Service, _ *llm.Tool) bool {
		return s.hasTypes(PaymentType)
	},
	Tool: llm.Tool{Name: "payment_link",
		Description: "A link to pay the person through their own Stripe account, for a quote, a deposit or a bill: what for, the amount and currency. It is put to them first; once made, put the link in an email or a message. Paid is marked as Stripe says, so an action can follow it (payment, paid=true).",
		Schema: obj(map[string]any{
			"what":     map[string]any{"type": "string"},
			"amount":   map[string]any{"type": "number", "description": "In whole units: 60 or 60.50."},
			"currency": map[string]any{"type": "string", "description": "Three letters; gbp unless said."},
		}, "what", "amount")}}

type paymentArgs struct {
	What     string  `json:"what"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

func (s *Service) paymentOf(raw json.RawMessage) (paymentArgs, error) {
	var p paymentArgs
	json.Unmarshal(raw, &p)
	p.What, p.Currency = trim.Line(p.What, 200), strings.ToLower(strings.TrimSpace(p.Currency))
	if p.Currency == "" {
		p.Currency = "gbp"
	}
	switch {
	case p.What == "" || p.Amount <= 0:
		return p, fmt.Errorf("say what it is for and how much")
	case len(p.Currency) != 3:
		return p, fmt.Errorf("say the currency as three letters, such as gbp")
	case p.Amount > s.paymentsMax():
		return p, fmt.Errorf("%s is above the most one link may ask for (%s, set_setting payments.max); nothing was put to the person", money(p.Amount, p.Currency), fmt.Sprint(s.paymentsMax()))
	case s.StripeKey != nil && s.StripeKey() == "":
		return p, fmt.Errorf("no Stripe account is connected: the person connects theirs on Payments")
	}
	return p, nil
}

func (s *Service) paymentLink(raw json.RawMessage) toolResult {
	p, err := s.paymentOf(raw)
	if err == nil && (s.StripeKey == nil || s.StripeKey() == "") {
		err = fmt.Errorf("no Stripe account is connected: the person connects theirs on Payments")
	}
	if err != nil {
		return fail("%v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, link, err := s.stripe().Link(ctx, p.What, p.Amount, p.Currency)
	if err != nil {
		return fail("no link made: %v", err)
	}
	rec, err := s.put(records.Op{Type: PaymentType, After: map[string]any{"what": p.What, "amount": p.Amount, "currency": p.Currency, "link": link, "stripe": id}})
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{text: fmt.Sprintf("Made: %s for %q, paid at %s (payment %s).", money(p.Amount, p.Currency), p.What, link, rec.ID),
		change: &records.Change{Action: "created", Component: PaymentType, ID: rec.ID, Detail: p.What}}
}

// paymentsMax is the most one link may ask for, in whole units: what the
// owner set, or 500, so a slip of a zero is refused, not asked.
func (s *Service) paymentsMax() float64 {
	if n, err := strconv.ParseFloat(strings.TrimSpace(s.setting("payments.max")), 64); err == nil && n > 0 {
		return n
	}
	return 500
}

func (s *Service) stripe() stripe.Account {
	a := stripe.Account{Key: s.StripeKey()}
	if s.Automate != nil {
		a.HTTP = s.Automate.HTTP
	}
	return a
}

// money is an amount as a person reads it: 60.00 GBP.
func money(amount float64, currency string) string {
	return fmt.Sprintf("%.2f %s", amount, strings.ToUpper(currency))
}

// FollowPayments marks each payment Stripe says is paid, and tells the
// owner. It is a background job's work.
func (s *Service) FollowPayments(ctx context.Context) error {
	if !s.hasTypes(PaymentType) || s.StripeKey == nil || s.StripeKey() == "" {
		return nil
	}
	recs, err := s.Store.List(PaymentType, store.ListOptions{})
	if err != nil {
		return err
	}
	for _, r := range recs {
		id, _ := r.Fields["stripe"].(string)
		if paid, _ := r.Fields["paid"].(bool); paid || id == "" || s.Now().Sub(r.CreatedAt) > 90*24*time.Hour {
			continue
		}
		amount, _ := r.Fields["amount"].(float64)
		if yes, err := s.stripe().Paid(ctx, id, amount, fmt.Sprint(r.Fields["currency"])); err != nil || !yes {
			continue
		}
		done, err := s.Apply(records.Op{Type: PaymentType, ID: r.ID, After: map[string]any{"paid": true}})
		if err != nil {
			continue
		}
		what, _ := r.Fields["what"].(string)
		said := fmt.Sprintf("Paid: %s for %q.", money(amount, fmt.Sprint(r.Fields["currency"])), what)
		records.Record(s.Store, "system", records.Change{Action: "paid", Component: PaymentType, ID: r.ID, Detail: what, Ops: done})
		s.Say(said)
		if s.Automate != nil && s.Automate.Tell != nil {
			s.Automate.Tell("You were paid", said, "/t/"+PaymentType+"/"+r.ID)
		}
	}
	return nil
}

// stripeLink finds addresses to pay at, as Stripe gives them.
var stripeLink = regexp.MustCompile(`https?://(?:buy|checkout)\.stripe\.com/[^\s<>()"']+`)

// paymentsIn says, from the payments kept here, what each link to pay in
// some words asks for, so the owner reads the amount Stripe will take and
// not what the words around it say; a link to pay that was not made here
// is refused, so none goes out in the owner's name unseen.
func (s *Service) paymentsIn(text string) ([]string, error) {
	found := stripeLink.FindAllString(text, -1)
	if len(found) == 0 {
		return nil, nil
	}
	recs, _ := s.Store.List(PaymentType, store.ListOptions{})
	var lines []string
	for _, link := range found {
		var p *store.Record
		for _, r := range recs {
			if r.Fields["link"] == link {
				p = r
			}
		}
		if p == nil {
			return nil, fmt.Errorf("it carries a link to pay that was not made here (%s); only a link made with payment_link goes out", link)
		}
		amount, _ := p.Fields["amount"].(float64)
		state := "not paid yet"
		if paid, _ := p.Fields["paid"].(bool); paid {
			state = "already paid"
		}
		lines = append(lines, fmt.Sprintf("It carries a link to pay %s for %q (%s), whatever the words say.", money(amount, fmt.Sprint(p.Fields["currency"])), p.Fields["what"], state))
	}
	return lines, nil
}
