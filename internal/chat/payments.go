package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	Doing: saying("Making a link to pay you"),
	Asks: func(s *Service, _ toolArgs, call llm.ToolCall) (toolResult, bool) {
		p, err := s.paymentOf(call.Args)
		if err != nil {
			return fail("%v", err), true
		}
		return s.ask(question{fmt.Sprintf("Make a link to pay you %s for %q?", Money(p.Amount, p.Currency), p.What),
			fmt.Sprintf("Stripe would make a link that takes one payment of %s into your Stripe account, then closes. Check the amount: it is what they would pay.", Money(p.Amount, p.Currency)),
			"Make a " + Money(p.Amount, p.Currency) + " link", "Don't"},
			map[string]any{"tool": "payment_link", "what": p.What, "amount": p.Amount, "currency": p.Currency}), true
	},
	Run: func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.paymentLink(call.Args) },
	Offered: func(s *Service, _ *llm.Tool) bool {
		return s.hasTypes(PaymentType)
	},
	Tool: llm.Tool{Name: "payment_link",
		Description: "A link to pay the person through their own Stripe account, for a quote, a deposit or a bill: what for, the amount and currency. To ask someone by email, use send_email's pay instead: it makes their link and sends it in one yes. This makes a link alone, put to them first, for a message elsewhere. Paid is marked as Stripe says, so an action can follow it (payment, paid=true).",
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
	// Rounded as Stripe takes it, so the question says what is paid.
	scale := math.Pow10(stripe.Decimals(p.Currency))
	p.Amount = math.Round(p.Amount*scale) / scale
	most, hasMax := s.paymentsMax(p.Currency)
	switch {
	case p.What == "" || p.Amount <= 0:
		return p, fmt.Errorf("say what it is for and how much")
	case len(p.Currency) != 3:
		return p, fmt.Errorf("say the currency as three letters, such as gbp")
	case !hasMax:
		return p, fmt.Errorf("payments.max sets no most for %s (it is %q); nothing was put to the person, who can set one for it", strings.ToUpper(p.Currency), s.setting("payments.max"))
	case p.Amount > most:
		return p, fmt.Errorf("%s is above the most one link may ask for (%s, set_setting payments.max); nothing was put to the person", Money(p.Amount, p.Currency), Money(most, p.Currency))
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
	rec, err := s.makePayment(p, "")
	if err != nil {
		return fail("%v", err)
	}
	link, _ := rec.Fields["link"].(string)
	return toolResult{text: fmt.Sprintf("Made: %s for %q, paid at %s (payment %s).", Money(p.Amount, p.Currency), p.What, link, rec.ID),
		change: &records.Change{Action: "created", Component: PaymentType, ID: rec.ID, Detail: p.What}}
}

// makePayment has Stripe make a link that takes one payment, and keeps it
// as a payment, with whom it was asked of when it goes in an email.
func (s *Service) makePayment(p paymentArgs, of string) (*store.Record, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, link, err := s.stripe().Link(ctx, p.What, p.Amount, p.Currency)
	if err != nil {
		return nil, fmt.Errorf("no link made: %v", err)
	}
	fields := map[string]any{"what": p.What, "amount": p.Amount, "currency": p.Currency, "link": link, "stripe": id}
	if of != "" {
		fields["asked_of"] = of
	}
	return s.put(records.Op{Type: PaymentType, After: fields})
}

// paymentsMax is the most one link may ask for in a currency, in whole
// units, so a slip of a zero is refused, not asked: what the owner set for
// that currency, or for any (a number alone), or 500 when they set none.
// It is false when they named currencies and not this one.
func (s *Service) paymentsMax(currency string) (float64, bool) {
	set := strings.TrimSpace(s.setting("payments.max"))
	if set == "" {
		return 500, true
	}
	most, any := 0.0, false
	for _, part := range strings.Split(set, ",") {
		f := strings.Fields(part)
		if len(f) == 0 {
			continue
		}
		n, err := strconv.ParseFloat(f[0], 64)
		switch {
		case err != nil || n <= 0:
		case len(f) == 1:
			most, any = n, true
		case strings.EqualFold(f[1], currency):
			return n, true
		}
	}
	return most, any
}

func (s *Service) stripe() stripe.Account {
	a := stripe.Account{Key: s.StripeKey()}
	if s.Automate != nil {
		a.HTTP = s.Automate.HTTP
	}
	return a
}

// Money is an amount as a person reads it, or hears it read: 60.00 GBP,
// 3000 JPY.
func Money(amount float64, currency string) string {
	return fmt.Sprintf("%.*f %s", stripe.Decimals(currency), amount, strings.ToUpper(currency))
}

// A payment is followed for 90 days. One not paid in 14 is marked late,
// and the owner told once, with the day it stops being followed, so money
// missing is never missed in silence.
const (
	paymentLate     = 14 * 24 * time.Hour
	paymentFollowed = 90 * 24 * time.Hour
)

// FollowPayments marks each payment Stripe says is paid, and tells the
// owner; and each one late, and tells them that. It is a background job's
// work, and Stripe not answering is its error, so the job says so.
func (s *Service) FollowPayments(ctx context.Context) error {
	if !s.hasTypes(PaymentType) || s.StripeKey == nil || s.StripeKey() == "" {
		return nil
	}
	recs, err := s.Store.List(PaymentType, store.ListOptions{})
	if err != nil {
		return err
	}
	var failed error
	for _, r := range recs {
		id, _ := r.Fields["stripe"].(string)
		age := s.Now().Sub(r.CreatedAt)
		if paid, _ := r.Fields["paid"].(bool); paid || id == "" || age > paymentFollowed {
			continue
		}
		amount, _ := r.Fields["amount"].(float64)
		what, _ := r.Fields["what"].(string)
		money := Money(amount, fmt.Sprint(r.Fields["currency"]))
		yes, err := s.stripe().Paid(ctx, id, amount, fmt.Sprint(r.Fields["currency"]))
		if err != nil {
			failed = err
			continue
		}
		if !yes {
			if late, _ := r.Fields["late"].(bool); !late && age > paymentLate {
				s.late(r, money, what)
			}
			continue
		}
		done, err := s.Apply(records.Op{Type: PaymentType, ID: r.ID, After: map[string]any{"paid": true, "late": false}})
		if err != nil {
			continue
		}
		said := fmt.Sprintf("Paid: %s for %q.", money, what)
		records.Record(s.Store, "system", records.Change{Action: "paid", Component: PaymentType, ID: r.ID, Detail: what, Ops: done})
		s.Say(said)
		if s.Automate != nil && s.Automate.Tell != nil {
			s.Automate.Tell("You were paid", said, "/t/"+PaymentType+"/"+r.ID)
		}
	}
	if failed != nil {
		return fmt.Errorf("Stripe did not say whether payments are paid: %v", failed)
	}
	return nil
}

// late marks a payment not paid in time, and tells the owner.
func (s *Service) late(r *store.Record, money, what string) {
	done, err := s.Apply(records.Op{Type: PaymentType, ID: r.ID, After: map[string]any{"late": true}})
	if err != nil {
		return
	}
	of := ""
	if who, _ := r.Fields["asked_of"].(string); who != "" {
		of = " asked of " + who
	}
	said := fmt.Sprintf("Not paid yet: %s for %q%s, %d days ago. Sameway looks for it until %s.",
		money, what, of, int(s.Now().Sub(r.CreatedAt).Hours()/24), r.CreatedAt.Add(paymentFollowed).Format("2 January"))
	records.Record(s.Store, "system", records.Change{Action: "marked late", Component: PaymentType, ID: r.ID, Detail: what, Ops: done})
	s.Say(said)
	if s.Automate != nil && s.Automate.Tell != nil {
		s.Automate.Tell("Not paid yet", said, "/t/"+PaymentType+"/"+r.ID)
	}
}

// stripeLink finds addresses to pay at, as Stripe gives them, in any
// case, as a mail client opens them.
var stripeLink = regexp.MustCompile(`(?i)https?://(?:buy|checkout|invoice|pay)\.stripe\.com/[^\s<>()"']+`)

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
		link = strings.TrimRight(link, ".,;:!?") // the end of a sentence, not the link
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
		lines = append(lines, fmt.Sprintf("It carries a link to pay %s for %q (%s), whatever the words say.", Money(amount, fmt.Sprint(p.Fields["currency"])), p.Fields["what"], state))
	}
	return lines, nil
}
