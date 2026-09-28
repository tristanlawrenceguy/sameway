// Package discover finds what is on the local network that announces
// itself: printers, speakers, hubs, brokers, anything with mDNS (Bonjour,
// Avahi). One multicast question asks for every service type, one more
// per type asks who offers it, and the answers are gathered for a few
// seconds. Nothing is installed and nothing is sent beyond the local
// network.
package discover

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// Service is one thing found.
type Service struct {
	Type string // _mqtt._tcp, _hap._tcp, _http._tcp
	Name string // as announced, without the type
	Host string
	Port int
	Addr string
	Text []string
}

// Known says what a service type usually is, for a person reading the list.
var Known = map[string]string{
	"_mqtt._tcp":            "MQTT broker",
	"_hap._tcp":             "HomeKit accessory",
	"_googlecast._tcp":      "Chromecast or Google speaker",
	"_airplay._tcp":         "AirPlay",
	"_raop._tcp":            "AirPlay speaker",
	"_ipp._tcp":             "printer",
	"_printer._tcp":         "printer",
	"_http._tcp":            "web page",
	"_hue._tcp":             "Philips Hue bridge",
	"_spotify-connect._tcp": "Spotify Connect",
	"_sonos._tcp":           "Sonos",
	"_esphomelib._tcp":      "ESPHome device",
	"_home-assistant._tcp":  "Home Assistant",
	"_ssh._tcp":             "SSH",
	"_smb._tcp":             "file share",
	"_workstation._tcp":     "computer",
}

var group = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}

// Browse asks the local network what is there and returns what answered
// within the wait, grouped and sorted. types limits the question to
// those service types; empty asks for every type first.
func Browse(ctx context.Context, wait time.Duration, types ...string) ([]Service, error) {
	conn, err := net.ListenMulticastUDP("udp4", nil, group)
	if err != nil {
		return nil, fmt.Errorf("could not listen for answers on the local network: %w", err)
	}
	defer conn.Close()
	conn.SetReadBuffer(1 << 20)
	send, err := net.DialUDP("udp4", nil, group)
	if err != nil {
		return nil, err
	}
	defer send.Close()

	asked := map[string]bool{}
	ask := func(name string, t dnsmessage.Type) {
		if asked[name] {
			return
		}
		asked[name] = true
		if q, err := question(name, t); err == nil {
			send.Write(q)
		}
	}
	if len(types) == 0 {
		ask("_services._dns-sd._udp.local.", dnsmessage.TypePTR)
	}
	for _, t := range types {
		ask(strings.TrimSuffix(t, ".")+".local.", dnsmessage.TypePTR)
	}

	found := map[string]*Service{}
	hosts := map[string]string{}
	deadline := time.Now().Add(wait)
	buf := make([]byte, 65536)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		var p dnsmessage.Parser
		if _, err := p.Start(buf[:n]); err != nil {
			continue
		}
		p.SkipAllQuestions()
		// Answers, then additionals: a responder puts the SRV, TXT and A
		// records for a service in the additional section.
		additional := false
		next := func() (dnsmessage.Resource, error) {
			if !additional {
				rr, err := p.Answer()
				if err != dnsmessage.ErrSectionDone {
					return rr, err
				}
				additional = true
				p.SkipAllAuthorities()
			}
			return p.Additional()
		}
		for {
			rr, err := next()
			if err != nil {
				break
			}
			name := strings.TrimSuffix(rr.Header.Name.String(), ".")
			switch body := rr.Body.(type) {
			case *dnsmessage.PTRResource:
				target := strings.TrimSuffix(body.PTR.String(), ".")
				if name == "_services._dns-sd._udp.local" {
					ask(target+".", dnsmessage.TypePTR)
					continue
				}
				svc := entry(found, target, name)
				_ = svc
				ask(target+".", dnsmessage.TypeSRV)
			case *dnsmessage.SRVResource:
				svc := entry(found, name, "")
				svc.Host = strings.TrimSuffix(body.Target.String(), ".")
				svc.Port = int(body.Port)
				if a, ok := hosts[svc.Host]; ok {
					svc.Addr = a
				}
			case *dnsmessage.TXTResource:
				svc := entry(found, name, "")
				for _, t := range body.TXT {
					if t != "" {
						svc.Text = append(svc.Text, t)
					}
				}
			case *dnsmessage.AResource:
				ip := net.IP(body.A[:]).String()
				hosts[name] = ip
				for _, svc := range found {
					if svc.Host == name && svc.Addr == "" {
						svc.Addr = ip
					}
				}
			}
		}
	}
	var out []Service
	for _, svc := range found {
		if svc.Type == "" {
			continue
		}
		out = append(out, *svc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// entry is the service for an instance name such as
// "Kitchen._hap._tcp.local", made when new. The type is what follows the
// instance, or the type given when the PTR named it.
func entry(found map[string]*Service, instance, typ string) *Service {
	instance = strings.TrimSuffix(instance, ".")
	svc, ok := found[instance]
	if !ok {
		svc = &Service{}
		found[instance] = svc
	}
	if typ == "" {
		if i := strings.Index(instance, "._"); i > 0 {
			typ = strings.TrimSuffix(instance[i+1:], ".local")
		}
	}
	if svc.Type == "" {
		svc.Type = strings.TrimSuffix(typ, ".local")
	}
	if svc.Name == "" {
		svc.Name = strings.TrimSuffix(strings.TrimSuffix(instance, "."+svc.Type+".local"), ".local")
		svc.Name = strings.ReplaceAll(svc.Name, `\032`, " ")
	}
	return svc
}

// question is one mDNS query for a name and type.
func question(name string, t dnsmessage.Type) ([]byte, error) {
	n, err := dnsmessage.NewName(name)
	if err != nil {
		return nil, err
	}
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{})
	b.StartQuestions()
	if err := b.Question(dnsmessage.Question{Name: n, Type: t, Class: dnsmessage.ClassINET}); err != nil {
		return nil, err
	}
	return b.Finish()
}

// Describe is a service in a line: what it is, its name, and where.
func Describe(s Service) string {
	what := Known[s.Type]
	if what == "" {
		what = s.Type
	}
	where := s.Host
	if s.Addr != "" {
		where = s.Addr
	}
	if s.Port > 0 {
		where += fmt.Sprintf(":%d", s.Port)
	}
	return fmt.Sprintf("%-28s %-32s %s", what, s.Name, where)
}
