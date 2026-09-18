package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A workspace among others. Every workspace this machine has opened is
// known, and each runs as its own server on its own address, so several
// can be open side by side. From any of them a person can open another,
// start one that is not running, make a new blank one, copy this one, or
// delete this one. The command line provides what a page cannot do by
// itself: starting another server, and stopping this one.

// Fleet is what the command line lends the page for the workspaces it
// serves alongside this one.
type Fleet struct {
	// Launch starts a server for the workspace at dir on addr and returns
	// once it is started, not once it is ready.
	Launch func(dir, addr string) error
	// Exit stops this server, once the response in hand is written.
	Exit func()
}

// WithFleet gives the server the means to start and stop workspaces.
func (s *Server) WithFleet(f *Fleet) *Server {
	s.fleet = f
	return s
}

// place is one workspace as the page shows it.
type place struct {
	Dir, Name, URL string
	Running        bool
}

// others are the known workspaces other than this one, with whether each
// is running and where.
func (s *Server) others() []place {
	var out []place
	for _, k := range workspace.KnownWorkspaces() {
		if sameDir(k.Dir, s.app.Workspace.Dir) {
			continue
		}
		ws, err := workspace.Load(k.Dir)
		if err != nil {
			continue
		}
		p := place{Dir: k.Dir, Name: ws.Config.Name}
		p.URL, p.Running = running(k)
		out = append(out, p)
	}
	return out
}

// running says whether a known workspace has a server at the address it
// was last seen on: the server there must describe that very workspace,
// since two workspaces can be configured for the same port.
func running(k workspace.Known) (url string, ok bool) {
	if k.Addr == "" {
		return "", false
	}
	client := &http.Client{Timeout: 400 * time.Millisecond}
	resp, err := client.Get("http://" + k.Addr + "/api/describe/routes")
	if err != nil {
		return "", false
	}
	resp.Body.Close()
	resp, err = client.Get("http://" + k.Addr + "/api/describe")
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	var d struct {
		Dir string `json:"dir"`
	}
	if json.NewDecoder(resp.Body).Decode(&d) != nil || !sameDir(d.Dir, k.Dir) {
		return "", false
	}
	return "http://" + k.Addr + "/", true
}

func sameDir(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// start opens the workspace at dir: its running server, or a new one on
// its own address, waited for until it answers.
func (s *Server) start(dir string) (string, error) {
	for _, k := range workspace.KnownWorkspaces() {
		if sameDir(k.Dir, dir) {
			if url, ok := running(k); ok {
				return url, nil
			}
		}
	}
	if s.fleet == nil || s.fleet.Launch == nil {
		return "", errors.New("starting another workspace from here needs sameway open; run it for that workspace yourself")
	}
	addr, err := chooseAddr(dir)
	if err != nil {
		return "", err
	}
	if err := s.fleet.Launch(dir, addr); err != nil {
		return "", err
	}
	if err := workspace.Remember(dir, addr); err != nil {
		return "", err
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if url, ok := running(workspace.Known{Dir: dir, Addr: addr}); ok {
			return url, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", fmt.Errorf("the workspace at %s was started but did not answer at %s", dir, addr)
}

// chooseAddr is where a workspace should listen: the address in its own
// workspace.yaml when that is free, otherwise a free port on this
// machine, since every workspace is configured for the same port at first.
func chooseAddr(dir string) (string, error) {
	ws, err := workspace.Load(dir)
	if err != nil {
		return "", err
	}
	if l, err := net.Listen("tcp", ws.Config.Server.Addr); err == nil {
		l.Close()
		return ws.Config.Server.Addr, nil
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}
