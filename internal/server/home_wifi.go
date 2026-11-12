package server

import (
	"encoding/json"
	"errors"
	"log"
	"maps"
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
// is not answers nothing). Away means a quarter of an hour of knocks
// that found nothing, so walking out of reach of the router for a minute
// rings nothing. Only knocks count: while the computer is asleep, off or
// just started it is not looking, so a phone at home all night does not
// ring anything when the computer wakes. The address is kept, so the
// knocks go on after a restart. No location leaves the house: the
// computer only notices the phone on its own network.

// awayAfter is how long the knocks find nothing before the phone counts
// as away.
const awayAfter = 15 * time.Minute

// lookGap is a pause between knocks that means the computer was not
// looking: asleep, off, or Sameway stopped.
const lookGap = 2 * time.Minute

type homeWatch struct {
	mu     sync.Mutex
	at     map[string]string // a phone's last address on the Wi-Fi, by device
	looked time.Time         // the last knock
	missed time.Time         // since when every knock found nothing; zero while seen or not known
	knock  func(ip string) bool
}

// WithHomeKnock sets how a phone's address is knocked at: a test's own.
func (s *Server) WithHomeKnock(knock func(ip string) bool) *Server {
	s.home.knock = knock
	return s
}

// homeAt is the phones' addresses, read from the store after a restart.
// The caller holds s.home.mu.
func (s *Server) homeAt() map[string]string {
	if s.home.at == nil {
		s.home.at = map[string]string{}
		json.Unmarshal([]byte(s.app.Store.Meta("home:at")), &s.home.at)
	}
	return s.home.at
}

// seenHome is the owner's phone making a request over the Wi-Fi.
func (s *Server) seenHome(d lanDevice, remote string) {
	if d.Person != "" || d.Only != "" {
		return
	}
	if host, _, err := net.SplitHostPort(remote); err == nil {
		s.home.mu.Lock()
		if at := s.homeAt(); at[d.ID] != host {
			at[d.ID] = host
			b, _ := json.Marshal(at)
			s.app.Store.SetMeta("home:at", string(b))
		}
		s.home.mu.Unlock()
	}
	s.backHome()
}

// lookHome knocks at each of the owner's phones where it was last seen.
func (s *Server) lookHome() {
	now := s.now()
	s.home.mu.Lock()
	if now.Sub(s.home.looked) > lookGap {
		s.home.missed = time.Time{} // not looking until now: whether it left is not known
	}
	s.home.looked = now
	at := maps.Clone(s.homeAt())
	knock := s.home.knock
	s.home.mu.Unlock()
	var ips []string
	for _, d := range s.lanDevices() {
		if ip, ok := at[d.ID]; ok && d.Person == "" && d.Only == "" {
			ips = append(ips, ip) // a phone removed is knocked at no more
		}
	}
	if knock == nil {
		knock = knockAt
	}
	for _, ip := range ips {
		if knock(ip) {
			s.backHome()
			return
		}
	}
	s.home.mu.Lock()
	if s.home.missed.IsZero() && len(ips) > 0 {
		s.home.missed = now
	}
	s.home.mu.Unlock()
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

// backHome is the phone seen now: when the knocks had missed it long
// enough, it rings what waits for it.
func (s *Server) backHome() {
	now := s.now()
	s.home.mu.Lock()
	away := !s.home.missed.IsZero() && now.Sub(s.home.missed) >= awayAfter
	s.home.missed = time.Time{}
	s.home.mu.Unlock()
	if away {
		s.ringHome(now)
	}
}

// ringHome rings the reminders set for getting home whose time has come,
// as a time rings one.
func (s *Server) ringHome(now time.Time) {
	t, ok := s.app.Types.Get(ReminderType)
	if !ok {
		return
	}
	recs, err := query.Filter(s.app.Store, t, []string{"state=set", "home=true"}, "at", 0, now)
	if err != nil {
		return
	}
	rang := 0
	for _, rec := range recs {
		at, _ := rec.Fields["at"].(string)
		if ts, err := time.Parse(time.RFC3339, at); err == nil && ts.After(now) {
			continue // it waits from its time: "when I get home on Friday"
		}
		if _, err := records.ApplyOps(s.app.Store, records.Op{Type: ReminderType, ID: rec.ID, After: map[string]any{"state": "rang"}}); err != nil {
			log.Printf("home: %v", err)
			continue
		}
		rang++
		records.Record(s.app.Store, "system", records.Change{Action: "rang", Component: ReminderType, ID: rec.ID, Detail: s.title(t, rec)})
		if s.notify != nil {
			text, url := blocks.RingWords(s.app.Store, rec, now)
			go s.notify(s.title(t, rec), text, s.linkTo(url))
		}
	}
	if rang > 0 {
		s.Changed()
	}
}
