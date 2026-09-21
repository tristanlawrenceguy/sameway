package notify

import (
	"errors"
	"runtime"
	"strings"
	"testing"
)

type call struct {
	env  []string
	name string
	args []string
}

// A ring is told two ways: the system's own notification, with the words
// in the environment rather than on the command line, and a command with
// the words where its placeholders stand, a phrase in quotes staying one
// argument. Each way is tried even when the other fails.
func TestARingIsToldEveryWay(t *testing.T) {
	var calls []call
	n := Notifier{Desktop: true, Command: `curl -d "Reminder: {title}" -H "Click: {url}" ntfy.sh/mine`, Run: func(env []string, name string, args ...string) error {
		calls = append(calls, call{env, name, args})
		if name == "curl" {
			return errors.New("no network")
		}
		return nil
	}}
	err := n.Send("Tea", "It is time.", "http://127.0.0.1:8080/t/reminder/r1")
	if err == nil || !strings.Contains(err.Error(), "notify.command") || !strings.Contains(err.Error(), "no network") {
		t.Errorf("a failing way is reported, by name: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("both ways are tried, got %d calls", len(calls))
	}
	desk, cmd := calls[0], calls[1]
	want := map[string]string{"windows": "powershell", "darwin": "osascript"}[runtime.GOOS]
	if want == "" {
		want = "notify-send"
	}
	if desk.name != want || !strings.Contains(strings.Join(desk.env, " "), "SAMEWAY_TITLE=Tea") {
		t.Errorf("the desktop notification is the system's own, with the words in the environment: %s %v", desk.name, desk.env)
	}
	if cmd.name != "curl" || strings.Join(cmd.args, "|") != "-d|Reminder: Tea|-H|Click: http://127.0.0.1:8080/t/reminder/r1|ntfy.sh/mine" {
		t.Errorf("the command gets the words where its placeholders stand: %s %v", cmd.name, cmd.args)
	}

	quiet := Notifier{Run: func([]string, string, ...string) error { t.Error("nothing to run"); return nil }}
	if err := quiet.Send("Tea", "", ""); err != nil {
		t.Errorf("no ways is no error: %v", err)
	}
}

// The Windows script travels encoded, as PowerShell takes it: UTF-16LE
// in base64, which is the same length for the same script every time.
func TestTheToastScriptIsEncoded(t *testing.T) {
	e := encoded("Write-Host hi")
	if e != "VwByAGkAdABlAC0ASABvAHMAdAAgAGgAaQA=" {
		t.Errorf("UTF-16LE base64, got %s", e)
	}
}
