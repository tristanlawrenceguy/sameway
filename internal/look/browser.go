package look

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// A page with its scripts run needs a browser. Sameway brings none: it
// uses the Chrome, Edge or Chromium the machine already has, headless,
// through the DevTools protocol, the way the browser's own tools do. The
// page is read with the same reader and the same rules as without.

// ErrNoBrowser says there is no browser to run a page's scripts in.
var ErrNoBrowser = errors.New("reading a page with its scripts needs Chrome, Edge or Chromium, and none was found; install one, or set SAMEWAY_BROWSER to its program")

// FindBrowser is the program to run pages in: SAMEWAY_BROWSER when set,
// otherwise the first Chrome, Edge or Chromium in the usual places.
func FindBrowser() (string, error) {
	if p := os.Getenv("SAMEWAY_BROWSER"); p != "" {
		return p, nil
	}
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData")} {
			if root != "" {
				candidates = append(candidates,
					filepath.Join(root, `Google\Chrome\Application\chrome.exe`),
					filepath.Join(root, `Microsoft\Edge\Application\msedge.exe`),
					filepath.Join(root, `Chromium\Application\chrome.exe`))
			}
		}
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium"}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge", "chrome", "msedge"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", ErrNoBrowser
}

// browser is one headless browser, started for one reading and gone after.
type browser struct {
	cmd     *exec.Cmd
	dir     string
	conn    *websocket.Conn
	session string

	mu      sync.Mutex
	next    int
	waiting map[int]chan reply
	errors  []string
}

type reply struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// launch starts the browser with a page open and listened to.
func launch(ctx context.Context, program string) (*browser, error) {
	dir, err := os.MkdirTemp("", "sameway-look-")
	if err != nil {
		return nil, err
	}
	b := &browser{dir: dir, waiting: map[int]chan reply{}}
	b.cmd = exec.Command(program, "--headless=new", "--remote-debugging-port=0", "--user-data-dir="+dir,
		"--no-first-run", "--no-default-browser-check", "--disable-gpu", "--disable-extensions",
		"--window-size=1280,900", "about:blank")
	if err := b.cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("starting %s: %w", program, err)
	}
	// The browser writes the port it listens on, and its address, here.
	var port []string
	for deadline := time.Now().Add(20 * time.Second); ; {
		if raw, err := os.ReadFile(filepath.Join(dir, "DevToolsActivePort")); err == nil {
			if port = strings.Fields(string(raw)); len(port) == 2 {
				break
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			b.close()
			return nil, fmt.Errorf("%s did not open its DevTools port", program)
		}
		time.Sleep(50 * time.Millisecond)
	}
	b.conn, _, err = websocket.Dial(ctx, "ws://127.0.0.1:"+port[0]+port[1], nil)
	if err != nil {
		b.close()
		return nil, fmt.Errorf("reaching the browser: %w", err)
	}
	b.conn.SetReadLimit(256 << 20)
	go b.read()
	var target struct {
		TargetID string `json:"targetId"`
	}
	if err := b.call(ctx, "Target.createTarget", map[string]any{"url": "about:blank"}, &target); err != nil {
		b.close()
		return nil, err
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := b.call(ctx, "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, &attached); err != nil {
		b.close()
		return nil, err
	}
	b.session = attached.SessionID
	for _, domain := range []string{"Page.enable", "Runtime.enable", "Log.enable"} {
		if err := b.call(ctx, domain, nil, nil); err != nil {
			b.close()
			return nil, err
		}
	}
	return b, nil
}

// read hands each reply to whoever asked, and keeps what went wrong on
// the page: an exception, an error written to the console, a request
// that failed.
func (b *browser) read() {
	for {
		_, raw, err := b.conn.Read(context.Background())
		if err != nil {
			b.mu.Lock()
			for id, ch := range b.waiting {
				ch <- reply{Error: &struct {
					Message string `json:"message"`
				}{Message: "the browser went away"}}
				delete(b.waiting, id)
			}
			b.mu.Unlock()
			return
		}
		var msg struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			reply
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		if msg.Method != "" {
			if e := pageError(msg.Method, msg.Params); e != "" {
				b.mu.Lock()
				b.errors = append(b.errors, e)
				b.mu.Unlock()
			}
			continue
		}
		b.mu.Lock()
		ch := b.waiting[msg.ID]
		delete(b.waiting, msg.ID)
		b.mu.Unlock()
		if ch != nil {
			ch <- msg.reply
		}
	}
}

// pageError is the words of an event that says something went wrong.
func pageError(method string, params json.RawMessage) string {
	switch method {
	case "Runtime.exceptionThrown":
		var p struct {
			Details struct {
				Text      string `json:"text"`
				Exception struct {
					Description string `json:"description"`
				} `json:"exception"`
			} `json:"exceptionDetails"`
		}
		json.Unmarshal(params, &p)
		if d := p.Details.Exception.Description; d != "" {
			return strings.SplitN(d, "\n", 2)[0]
		}
		return p.Details.Text
	case "Runtime.consoleAPICalled":
		var p struct {
			Type string `json:"type"`
			Args []struct {
				Value       any    `json:"value"`
				Description string `json:"description"`
			} `json:"args"`
		}
		json.Unmarshal(params, &p)
		if p.Type != "error" {
			return ""
		}
		var words []string
		for _, a := range p.Args {
			if a.Value != nil {
				words = append(words, fmt.Sprint(a.Value))
			} else {
				words = append(words, strings.SplitN(a.Description, "\n", 2)[0])
			}
		}
		return "console: " + strings.Join(words, " ")
	case "Log.entryAdded":
		var p struct {
			Entry struct {
				Level string `json:"level"`
				Text  string `json:"text"`
				URL   string `json:"url"`
			} `json:"entry"`
		}
		json.Unmarshal(params, &p)
		// The icon a browser asks every site for is not the page's doing.
		if p.Entry.Level != "error" || strings.HasSuffix(p.Entry.URL, "/favicon.ico") {
			return ""
		}
		return strings.TrimSpace(p.Entry.Text + " " + p.Entry.URL)
	}
	return ""
}

// call sends one DevTools command to the page and reads its result into out.
func (b *browser) call(ctx context.Context, method string, params any, out any) error {
	b.mu.Lock()
	b.next++
	id := b.next
	ch := make(chan reply, 1)
	b.waiting[id] = ch
	b.mu.Unlock()
	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if b.session != "" {
		msg["sessionId"] = b.session
	}
	raw, _ := json.Marshal(msg)
	if err := b.conn.Write(ctx, websocket.MessageText, raw); err != nil {
		return err
	}
	select {
	case r := <-ch:
		if r.Error != nil {
			return fmt.Errorf("%s: %s", method, r.Error.Message)
		}
		if out != nil && len(r.Result) > 0 {
			return json.Unmarshal(r.Result, out)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// pageErrors is what has gone wrong on the page so far.
func (b *browser) pageErrors() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.errors...)
}

func (b *browser) close() {
	if b.conn != nil {
		b.conn.Close(websocket.StatusNormalClosure, "")
	}
	if b.cmd != nil && b.cmd.Process != nil {
		b.cmd.Process.Kill()
		b.cmd.Wait()
	}
	// The browser lets go of its profile a moment after it exits.
	for i := 0; i < 20; i++ {
		if os.RemoveAll(b.dir) == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
