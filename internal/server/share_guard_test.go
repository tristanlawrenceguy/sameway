package server

import (
	"net"
	"testing"
)

// The addresses a shared page is never read from are this computer's and
// its network's, written any way they can be; the internet's are read.
func TestNearbyIsThisComputerAndItsNetwork(t *testing.T) {
	t.Parallel()
	for _, addr := range []string{"127.0.0.1", "::1", "0.0.0.0", "0.1.2.3", "::", "::ffff:127.0.0.1", "10.0.0.1", "172.16.4.2",
		"192.168.1.1", "100.100.100.100", "169.254.169.254", "fe80::1", "fd00::1", "224.0.0.1", "ff02::1"} {
		if !nearby(net.ParseIP(addr)) {
			t.Errorf("%s is nearby and never read", addr)
		}
	}
	for _, addr := range []string{"93.184.216.34", "8.8.8.8", "2606:4700::1111", "172.32.0.1", "100.128.0.1"} {
		if nearby(net.ParseIP(addr)) {
			t.Errorf("%s is on the internet and read", addr)
		}
	}
}

// A link keeps what names its page and loses what follows who clicked it.
func TestAKeptLinkHasNoTracking(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"https://example.com/a?id=3&utm_source=x&UTM_Medium=y&fbclid=z#part": "https://example.com/a?id=3#part",
		"https://example.com/a?gclid=1":                                      "https://example.com/a",
		"http://example.com/?q=tea&page=2":                                   "http://example.com/?q=tea&page=2",
		"https://example.com/search?utm=keep":                                "https://example.com/search?utm=keep",
	} {
		u, err := keptLink(in)
		if err != nil || u.String() != want {
			t.Errorf("%s kept as %v (%v), want %s", in, u, err, want)
		}
	}
	for _, bad := range []string{"ftp://example.com", "javascript:alert(1)", "https://", "example.com"} {
		if _, err := keptLink(bad); err == nil {
			t.Errorf("%s is not a link to keep", bad)
		}
	}
}
