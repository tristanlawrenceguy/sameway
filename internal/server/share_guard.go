package server

import (
	"errors"
	"net"
	"net/http"
	"syscall"
	"time"
)

// A shared link is read by this computer, so a link (or a page it
// redirects to) must not make Sameway read what only this computer or
// its network should see: a router's settings, a printer, another
// program's page, the metadata a cloud machine answers on. Each address
// is checked as it is dialled, after the name is looked up, so a
// redirect or a name that changes what it points to is checked too
// (OWASP SSRF prevention). No proxy is used: a proxy would read the page
// past the check. The link itself is still kept, only its page unread.

var errNearby = errors.New("a page on this computer or its network is not read")

// pageReader is the client a shared page is read with, unless the app was
// given another (app.Options.Pages).
var pageReader = &http.Client{
	Timeout: pageWait,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{Timeout: 5 * time.Second, Control: func(_, address string, _ syscall.RawConn) error {
			host, _, _ := net.SplitHostPort(address)
			if ip := net.ParseIP(host); ip == nil || nearby(ip) {
				return errNearby
			}
			return nil
		}}).DialContext,
		ResponseHeaderTimeout: pageWait,
	},
}

// carrierNAT is the range carriers and Tailscale give the devices on a network
// (100.64.0.0/10), which is no more the internet than 192.168.
var carrierNAT = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// nearby says an address is this computer's or its network's, never the
// internet's.
func nearby(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil && (v4[0] == 0 || carrierNAT.Contains(v4)) {
		return true
	}
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast()
}

// pages is the client this server reads shared pages with.
func (s *Server) pages() *http.Client {
	if c := s.app.Options().Pages; c != nil {
		return c
	}
	return pageReader
}
