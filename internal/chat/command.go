package chat

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A command action runs a command line on the person's machine, as them.
// The assistant writes it when asked, so building stays a conversation;
// the person accepts it once, on the same card every other question uses,
// with the line in front of them, and from then on it is a plain button.
// A record can arrive from an agent, an import or a model that read the
// wrong page, which is why nothing runs before that one acceptance, and
// why changing the command line takes the acceptance away again.

// CommandTimeout bounds one run, so a hung command cannot hold a turn.
var CommandTimeout = 60 * time.Second

// Workdir is where commands run when the action names no folder: the
// workspace, set by the app.
var Workdir string

// command runs an accepted command, or asks the person to accept it.
func (s *Service) command(ctx context.Context, rec *store.Record, title string) toolResult {
	line, _ := rec.Fields["command"].(string)
	line = strings.TrimSpace(line)
	if line == "" {
		return fail("action %s has no command to run", title)
	}
	if accepted, _ := rec.Fields["accepted"].(string); accepted != line {
		r := s.propose("Run this on your machine, as you, whenever this button is pressed? "+line,
			map[string]any{"tool": "accept_action", "id": rec.ID})
		if !r.isErr {
			r.text = "the command has not been accepted yet, so nothing ran; " + r.text
		}
		return r
	}
	ctx, cancel := context.WithTimeout(ctx, CommandTimeout)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", line)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", line)
	}
	if folder, _ := rec.Fields["folder"].(string); folder != "" {
		cmd.Dir = folder
	} else if Workdir != "" {
		cmd.Dir = Workdir
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	answer := strings.TrimSpace(out.String())
	if len([]rune(answer)) > 2000 {
		answer = string([]rune(answer)[:2000])
	}
	code := 0
	if err != nil {
		code = -1
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		}
		if ctx.Err() != nil {
			answer = strings.TrimSpace(answer + "\n(stopped after " + CommandTimeout.String() + ")")
		}
	}
	res := toolResult{text: fmt.Sprintf("%s finished with exit code %d", title, code)}
	if answer != "" {
		res.text += ": " + answer
	}
	res.changes = append(res.changes, Change{Action: "ran", Component: ActionType, ID: rec.ID,
		Detail: fmt.Sprintf("%s (exit %d)", title, code), Href: "/t/" + ActionType + "/" + rec.ID})
	if code != 0 {
		res.isErr = true
		return res
	}
	if show, _ := rec.Fields["show"].(bool); show && answer != "" {
		if c, err := s.show(rec, title, answer); err == nil {
			res.changes = append(res.changes, c)
		} else {
			res.text += ". Could not put it on the canvas: " + err.Error()
		}
	}
	return res
}

// acceptAction is what a person's Yes on the card does: the command line
// as it stands is accepted for good, then run. The accepted line is kept,
// not a flag, so an edited command asks again.
func (s *Service) acceptAction(ctx context.Context, id string) toolResult {
	rec, err := s.Store.Get(ActionType, id)
	if err != nil {
		return fail("no action with id %s", id)
	}
	line, _ := rec.Fields["command"].(string)
	line = strings.TrimSpace(line)
	if _, err := s.Store.Update(ActionType, id, map[string]any{"accepted": line}); err != nil {
		return fail("could not accept action %s: %v", id, err)
	}
	rec.Fields["accepted"] = line
	title, _ := rec.Fields["title"].(string)
	return s.command(ctx, rec, title)
}
