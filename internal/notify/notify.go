// Package notify reaches a person outside the app. A reminder that rings
// while no page is open still has to reach someone: the machine sameway
// runs on can show a notification of its own, and a command can carry the
// news further, to a push service such as ntfy, to email, to a text,
// with {title}, {text} and {url} standing in its arguments for the words.
package notify

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"
)

// Notifier is how a ring is told beyond the page.
type Notifier struct {
	// Desktop shows a notification on this machine, the way the system
	// shows them: a toast on Windows, a banner on a Mac, notify-send on
	// Linux.
	Desktop bool
	// Command is a command line run for each ring, with {title}, {text}
	// and {url} replaced in its arguments. Empty runs nothing.
	Command string
	// Run runs a program; nil runs it for real. Tests put their own here.
	Run func(env []string, name string, args ...string) error
}

// Send tells a ring every way the notifier has. Each way is tried even
// when another fails, and every failure comes back in one error.
func (n Notifier) Send(title, text, url string) error {
	var errs []error
	if n.Desktop {
		if err := n.desktop(title, text); err != nil {
			errs = append(errs, fmt.Errorf("desktop notification: %w", err))
		}
	}
	if strings.TrimSpace(n.Command) != "" {
		args := fill(tokens(n.Command), map[string]string{"{title}": title, "{text}": text, "{url}": url})
		if len(args) > 0 {
			if err := n.run(nil, args[0], args[1:]...); err != nil {
				errs = append(errs, fmt.Errorf("notify.command: %w", err))
			}
		}
	}
	return errors.Join(errs...)
}

// desktop is the system's own notification, on each system in its way.
// The words travel in the environment, never in a command line, so no
// title can break out of one.
func (n Notifier) desktop(title, text string) error {
	env := []string{"SAMEWAY_TITLE=" + title, "SAMEWAY_TEXT=" + text}
	switch runtime.GOOS {
	case "windows":
		return n.run(env, "powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", encoded(toastScript))
	case "darwin":
		return n.run(env, "osascript", "-e", "on run argv", "-e", `display notification (item 2 of argv) with title (item 1 of argv) sound name "default"`, "-e", "end run", title, text)
	default:
		return n.run(env, "notify-send", "--app-name=sameway", title, text)
	}
}

// toastScript shows a Windows toast through the runtime the system has,
// under the identity PowerShell already holds, so nothing needs
// installing or registering. The reminder scenario keeps it on screen
// and sounds the reminder tone.
const toastScript = `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] > $null
$t = [System.Security.SecurityElement]::Escape($env:SAMEWAY_TITLE)
$b = [System.Security.SecurityElement]::Escape($env:SAMEWAY_TEXT)
$x = New-Object Windows.Data.Xml.Dom.XmlDocument
$x.LoadXml("<toast scenario='reminder'><visual><binding template='ToastGeneric'><text>$t</text><text>$b</text></binding></visual><audio src='ms-winsoundevent:Notification.Reminder'/></toast>")
$n = New-Object Windows.UI.Notifications.ToastNotification $x
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe').Show($n)`

// encoded is a script the way -EncodedCommand takes it: UTF-16LE, base64.
func encoded(script string) string {
	u := utf16.Encode([]rune(script))
	b := make([]byte, 0, len(u)*2)
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	return base64.StdEncoding.EncodeToString(b)
}

func (n Notifier) run(env []string, name string, args ...string) error {
	if n.Run != nil {
		return n.Run(env, name, args...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v: %s", name, err, strings.TrimSpace(tail(string(out), 300)))
	}
	return nil
}

// tokens splits a command line the way a shell would for plain words and
// double-quoted phrases, so a phrase with a space stays one argument.
func tokens(line string) []string {
	var out []string
	var cur strings.Builder
	quoted, have := false, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted, have = !quoted, true
		case r == ' ' && !quoted:
			if have {
				out = append(out, cur.String())
				cur.Reset()
				have = false
			}
		default:
			cur.WriteRune(r)
			have = true
		}
	}
	if have {
		out = append(out, cur.String())
	}
	return out
}

// fill puts the words in wherever a placeholder stands, whole or as part
// of an argument, so "Reminder: {title}" works as well as {title} alone.
func fill(args []string, values map[string]string) []string {
	pairs := make([]string, 0, len(values)*2)
	for k, v := range values {
		pairs = append(pairs, k, v)
	}
	rep := strings.NewReplacer(pairs...)
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = rep.Replace(a)
	}
	return out
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
