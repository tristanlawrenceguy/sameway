package server

import (
	"errors"
	"log"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// "Remind me when I get home." The computer is at home and the phone is
// not always: when the owner's phone is back on the home Wi-Fi after a
// while away, the reminders set to ring on getting home ring. The phone
// is seen two ways: it opens Sameway on the Wi-Fi, or, every half
// minute, the computer knocks at the address it last had there and it
// answers (an iPhone listens on port 62078 for syncing; any phone that
// is there answers a knock at a closed port by refusing it, and one that
// is not answers nothing). Back means seen again after a quarter of an
// hour unseen, so walking out of reach of the router for a minute does
// not ring anything; when it was last seen is kept, so a restart of the
// computer does not either. No location leaves the house: the computer
// only notices the phone on its own network.

// awayAfter is how long unseen before the phone counts as away.
const awayAfter = 15 * time.Minute

type homeWatch struct {
	mu    sync.Mutex
	at    map[string]string // a phone's last address on the Wi-Fi, by device
	knock func(ip string) bool
}

// WithHomeKnock sets how a phone's address is knocked at: a test's own.
func (s *Server) WithHomeKnock(knock func(ip string) bool) *Server {
	s.home.knock = knock
	return s
}

// seenHome is the owner's phone making a request over the Wi-Fi.
func (s *Server) seenHome(d lanDevice, remote string) {
	if d.Person != "" || d.Only != "" {
		return
	}
	if host, _, err := net.SplitHostPort(remote); err == nil {
		s.home.mu.Lock()
		if s.home.at == nil {
			s.home.at = map[string]string{}
		}
		s.home.at[d.ID] = host
		s.home.mu.Unlock()
	}
	s.backHome()
}

// lookHome knocks at each of the owner's phones where it was last seen.
func (s *Server) lookHome() {
	s.home.mu.Lock()
	var ips []string
	for _, ip := range s.home.at {
		ips = append(ips, ip)
	}
	knock := s.home.knock
	s.home.mu.Unlock()
	if knock == nil {
		knock = knockAt
	}
	for _, ip := range ips {
		if knock(ip) {
			s.backHome()
			return
		}
	}
}

// knockAt says whether a phone is at an address: it lets the computer
// in, or refuses it, which only something there can do.
func knockAt(ip string) bool {
	// Windows tries a refused knock again before it says so, which takes
	// it about two seconds.
	c, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "62078"), 4*time.Second)
	if err == nil {
		c.Close()
		return true
	}
	var errno syscall.Errno
	return errors.As(err, &errno) && (errno == syscall.ECONNREFUSED || errno == 10061) // 10061 is Windows' WSAECONNREFUSED
}

// backHome notes the phone seen now, and when it was away long enough,
// rings what waits for it.
func (s *Server) backHome() {
	now := s.now()
	last, _ := time.Parse(time.RFC3339, s.app.Store.Meta("home:seen"))
	if now.Sub(last) < time.Minute {
		return // seen just now: nothing changes
	}
	s.app.Store.SetMeta("home:seen", now.Format(time.RFC3339))
	if last.IsZero() || now.Sub(last) < awayAfter {
		return
	}
	s.ringHome(now)
}

// ringHome rings the reminders set for getting home, as a time rings one.
func (s *Server) ringHome(now time.Time) {
	t, ok := s.app.Types.Get(ReminderType)
	if !ok {
		return
	}
	recs, err := query.Filter(s.app.Store, t, []string{"state=set", "home=true"}, "at", 0, now)
	if err != nil {
		return
	}
	for _, rec := range recs {
		if _, err := records.ApplyOps(s.app.Store, records.Op{Type: ReminderType, ID: rec.ID, After: map[string]any{"state": "rang"}}); err != nil {
			log.Printf("home: %v", err)
			continue
		}
		records.Record(s.app.Store, "system", records.Change{Action: "rang", Component: ReminderType, ID: rec.ID, Detail: s.title(t, rec)})
		if s.notify != nil {
			text, url := blocks.RingWords(s.app.Store, rec, now)
			go s.notify(s.title(t, rec), text, s.linkTo(url))
		}
	}
	if len(recs) > 0 {
		s.Changed()
	}
}
