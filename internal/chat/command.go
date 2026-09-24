package chat

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A command action runs a program on the person's machine, as them. The
// assistant writes it when asked, so building stays a conversation; the
// person accepts it once, on the same card every other question uses,
// with the line in front of them, and from then on it is a plain button.
//
// The boundary around it: a command runs only a program the workspace
// has allowed by name (actions.allow in workspace.yaml), never through a
// shell, so the line is words and quoted phrases handed to that program
// and nothing else has meaning in it; it runs inside the workspace
// folder or a folder under it; and it stops after a minute. A record can
// arrive from an agent, an import or a model that read the wrong page,
// which is why nothing runs before the one acceptance, why changing the
// line takes the acceptance away, and why the allow list is the
// person's, kept where only they and their agents write.

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
	args := tokens(line)
	if len(args) == 0 {
		return fail("action %s has no program to run", title)
	}
	if !s.allowed(args[0]) {
		return fail("action %s would run %s, which this workspace has not allowed. Command actions run only programs named in actions.allow in workspace.yaml (allowed now: %s); add %s there, by asking or by editing the file, and the button will work", title, args[0], s.allowedWords(), programName(args[0]))
	}
	dir, err := s.folderFor(rec)
	if err != nil {
		return fail("action %s: %v", title, err)
	}
	if accepted, _ := rec.Fields["accepted"].(string); accepted != line {
		r := s.ask(question{"Let your " + quoted(title) + " button run a program on this computer?",
			"Pressing it would run " + line + " as you, with access to your files, every time, without asking again. What it does can't be undone from Sameway, so only say yes if you asked for this button.",
			"Yes, let it run", "No, don't"},
			map[string]any{"tool": "accept_action", "id": rec.ID})
		if !r.isErr {
			r.text = "the command has not been accepted yet, so nothing ran; " + r.text
		}
		return r
	}
	ctx, cancel := context.WithTimeout(ctx, CommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err = cmd.Run()
	answer := strings.TrimSpace(out.String())
	if len([]rune(answer)) > 2000 {
		answer = string([]rune(answer)[:2000])
	}
	code := 0
	if err != nil {
		code = -1
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else if answer == "" {
			answer = err.Error()
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

// allowed says whether a program is one the workspace lets actions run,
// by its name: curl, whether written as curl, curl.exe or a full path.
func (s *Service) allowed(program string) bool {
	name := programName(program)
	for _, a := range s.Allow {
		if programName(a) == name {
			return true
		}
	}
	return false
}

func (s *Service) allowedWords() string {
	if len(s.Allow) == 0 {
		return "none"
	}
	return strings.Join(s.Allow, ", ")
}

// programName is a program as the allow list names it: the base name,
// lower case, without .exe.
func programName(program string) string {
	name := strings.ToLower(filepath.Base(strings.Trim(program, `"`)))
	return strings.TrimSuffix(name, ".exe")
}

// folderFor is where an action runs: its folder when it names one, which
// must be the workspace or under it, and the workspace otherwise.
func (s *Service) folderFor(rec *store.Record) (string, error) {
	folder, _ := rec.Fields["folder"].(string)
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return Workdir, nil
	}
	if Workdir == "" {
		return folder, nil
	}
	if !filepath.IsAbs(folder) {
		folder = filepath.Join(Workdir, folder)
	}
	rel, err := filepath.Rel(Workdir, folder)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("the folder %s is outside the workspace; a command runs in the workspace or a folder under it", folder)
	}
	return folder, nil
}

// tokens splits a command line the way a shell would for plain words and
// double-quoted phrases, and nothing else: no pipes, no redirects, no
// variables, no second command. What is left is the program and its
// arguments, exactly.
func tokens(line string) []string {
	var out []string
	var cur strings.Builder
	quoted, have := false, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted, have = !quoted, true
		case (r == ' ' || r == '\t') && !quoted:
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
