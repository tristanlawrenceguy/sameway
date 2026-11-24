package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

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
	if !s.hasTypes(PaymentType) {
		return nil
	}
	if s.StripeKey == nil || s.StripeKey() == "" {
		if said := s.PaymentsWaiting(); said != "" {
			return fmt.Errorf("Stripe is disconnected, so %s: connect it again on Payments", said)
		}
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

// PaymentsWaiting says how many payments are not paid yet and still
// looked for, as "2 payments not yet paid are not looked for", or "" for
// none: what disconnecting Stripe leaves unwatched.
func (s *Service) PaymentsWaiting() string {
	if !s.hasTypes(PaymentType) {
		return ""
	}
	recs, _ := s.Store.List(PaymentType, store.ListOptions{})
	n := 0
	for _, r := range recs {
		id, _ := r.Fields["stripe"].(string)
		if paid, _ := r.Fields["paid"].(bool); !paid && id != "" && s.Now().Sub(r.CreatedAt) <= paymentFollowed {
			n++
		}
	}
	switch n {
	case 0:
		return ""
	case 1:
		return "1 payment not yet paid is not looked for"
	}
	return fmt.Sprintf("%d payments not yet paid are not looked for", n)
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
