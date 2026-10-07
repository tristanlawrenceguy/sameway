package cli

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
)

// lanServer answers on this computer's Wi-Fi address as well, on the same
// port, for phones paired on Workspaces (internal/server phone_lan.go):
// the handler it serves lets nothing past pairing but a paired phone.
type lanServer struct {
	mu   sync.Mutex
	srv  *http.Server
	base string
	port string
	h    http.Handler
}

// set starts or stops answering on the Wi-Fi.
func (l *lanServer) set(on bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !on {
		if l.srv != nil {
			l.srv.Shutdown(context.Background())
			l.srv, l.base = nil, ""
		}
		return nil
	}
	if l.srv != nil {
		return nil
	}
	ip, err := lanIP()
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(ip.String(), l.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.New("could not answer on the Wi-Fi at " + addr + ": " + err.Error())
	}
	l.srv = &http.Server{Handler: l.h}
	l.base = "http://" + addr
	go l.srv.Serve(ln)
	return nil
}

// Base is the Wi-Fi address while it answers there, "" when it does not.
func (l *lanServer) Base() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.base
}

// lanIP is this computer's address on its home network: a private IPv4
// address of an interface that is up, not a loopback or a tunnel.
func lanIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 || i.Flags&net.FlagPointToPoint != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok {
				if ip := n.IP.To4(); ip != nil && ip.IsPrivate() {
					return ip, nil
				}
			}
		}
	}
	return nil, errors.New("this computer is on no home network Sameway can find; connect it to the Wi-Fi first")
}
