package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// What the assistant does is either reversible, and then it just does it
// and the activity log can take it back, or it is not, and then it says so
// and asks first. The owner's rule: make everything reversible; if it
// cannot be, state it and ask.
//
// What cannot be taken back is what leaves the workspace: a program run
// on the machine, a request sent to an address, a message published to a
// device, and the settings that decide where the conversation and the
// person's secrets go, what may run, and who can reach the workspace. The
// assistant reads words other people wrote (mail, files, shared records),
// and those words must never be enough to make any of that happen. So the
// question is put by the code, from what the call would really do, never
// in the model's words, and a Yes runs exactly that call.
//
// A person pressing their own button is their own decision and is not
// asked again; a command still asks once, as it always has.

// A question is three things a person can act on: what is being asked,
// in everyday words; what would happen, named by what they know (their
// button, the website, what changes from what to what); and the two
// answers, saying what each does.
type question struct{ ask, detail, yes, no string }

// outward are the settings whose change cannot be taken back once it has
// acted, each with the question it puts, from the value it would take and
// the one it has now.
var outward = map[string]func(now, next string, s *Service) question{
	"tailnet.peers": func(now, next string, _ *Service) question {
		return question{"Keep this workspace in step with other computers?",
			fmt.Sprintf("The assistant wants this workspace to keep in step with %s (now: %s). Everything in it except the conversations with the assistant would be copied to them and kept the same both ways, and what they change comes here. Only say yes to computers you, or people you made hosts, run.", next, orNone(now)),
			"Yes, keep in step", "No, keep it as it is"}
	},
	"tailnet.name": func(now, next string, _ *Service) question {
		return question{"Open this workspace from your phone?",
			fmt.Sprintf("The assistant wants to put this workspace on your Tailscale network as %s, so your phone and your other devices signed in to Tailscale as you can open it from anywhere. Nobody else can. It needs a free Tailscale account, and the Tailscale app on your phone. This computer contacts Tailscale to join.", next),
			"Yes, turn it on", "No, only on this computer"}
	},
	"llm.base_url": func(now, next string, _ *Service) question {
		return question{"Send your conversations to a different AI service?",
			fmt.Sprintf("The assistant wants to change where its AI model runs, from %s to %s. From then on, everything you write here, with your notes and past messages, would be sent to %s. Only say yes if you set that service up yourself: what has been sent can't be taken back.", orNone(now), next, host(next)),
			"Yes, use " + host(next), "No, keep " + orNone(host(now))}
	},
	"llm.provider": func(now, next string, _ *Service) question {
		return question{"Switch the assistant to a different AI service?",
			fmt.Sprintf("The assistant wants to change which kind of AI service it uses, from %s to %s. From then on, your conversations, notes and past messages would go to that service. Only say yes if you asked for this: what has been sent can't be taken back.", orNone(now), next),
			"Yes, switch", "No, keep " + orNone(now)}
	},
	"llm.api_key_env": func(now, next string, s *Service) question {
		return question{"Let the assistant use a different saved password?",
			fmt.Sprintf("The assistant wants to sign in to its AI service with the secret saved on this computer as %s (now %s). That secret would be sent to %s with every message. Only say yes if %s is the key for that service.", next, orNone(now), orNone(host(s.setting("llm.base_url"))), next),
			"Yes, use " + next, "No, keep it as it is"}
	},
	"notify.command": func(now, next string, _ *Service) question {
		return question{"Run a program every time a reminder goes off?",
			fmt.Sprintf("The assistant wants each reminder to also run this on your computer, as you, with access to your files: %s. A program can do things Sameway can't undo, so only say yes if you asked for this.", next),
			"Yes, run it", "No, don't"}
	},
	"actions.allow": func(now, next string, _ *Service) question {
		return question{"Let your buttons run programs on this computer?",
			fmt.Sprintf("Buttons would be allowed to run: %s (now: %s). A program runs as you, with access to your files, and what it does can't be undone from Sameway.", next, orNone(now)),
			"Yes, allow them", "No, keep it as it is"}
	},
	"server.addr": func(now, next string, _ *Service) question {
		ask := "Change the address this workspace is served on?"
		if !privateAddr(next) {
			ask = "Open this workspace to other devices?"
		}
		return question{ask,
			fmt.Sprintf("From the next start, Sameway would listen at %s instead of %s. Anyone who can reach that address could read and change everything here, and there is no password. Only say yes if you know who can reach it.", next, orNone(now)),
			"Yes, change it", "No, keep " + orNone(now)}
	},
	"mcp.token_env": func(now, next string, _ *Service) question {
		return question{"Change the key other apps use to control this workspace?",
			fmt.Sprintf("Other apps would sign in with the secret saved on this computer as %s (now %s). Anyone who has that secret could read and change everything here.", next, orNone(now)),
			"Yes, change it", "No, keep it as it is"}
	},
	"mqtt.broker": func(now, next string, _ *Service) question {
		return question{"Send your devices' messages somewhere else?",
			fmt.Sprintf("Messages to and from your lights, plugs and other devices would go to %s instead of %s, from the next start. Only say yes if that is your own hub: what is sent there can't be taken back.", next, orNone(now)),
			"Yes, use " + host(next), "No, keep it as it is"}
	},
	"mqtt.username_env": mqttSecret,
	"mqtt.password_env": mqttSecret,
}

func mqttSecret(now, next string, s *Service) question {
	return question{"Sign in to your devices' hub with a different saved password?",
		fmt.Sprintf("The secret saved on this computer as %s (now %s) would be sent to %s.", next, orNone(now), orNone(s.setting("mqtt.broker"))),
		"Yes, use " + next, "No, keep it as it is"}
}

// askFirst puts an irreversible call to the person instead of making it.
// It answers false when the call is reversible and should simply run.
func (s *Service) askFirst(call string, id, key, value string) (toolResult, bool) {
	switch call {
	case "set_setting":
		put, ok := outward[key]
		// Taking the workspace off the tailnet, or no longer keeping in
		// step with anyone, sends nothing anywhere.
		if !ok || ((key == "tailnet.name" || key == "tailnet.peers") && strings.TrimSpace(value) == "") {
			return toolResult{}, false
		}
		q := put(s.setting(key), strings.TrimSpace(value), s)
		return s.ask(q, map[string]any{"tool": "set_setting", "key": key, "value": value}), true
	case "run_action":
		rec, err := s.Store.Get(ActionType, id)
		if err != nil {
			return toolResult{}, false // Run says there is no such action
		}
		title, _ := rec.Fields["title"].(string)
		press := "The assistant wants to press your " + quoted(title) + " button now."
		var q question
		switch kind, _ := rec.Fields["kind"].(string); kind {
		case "arrangement", "message":
			return toolResult{}, false // inside the workspace, and undone like any change
		case "command":
			line, _ := rec.Fields["command"].(string)
			line = strings.TrimSpace(line)
			// A command that would be refused, or that has not been
			// accepted and so asks on its own card, is left to Run.
			args := tokens(line)
			if len(args) == 0 || !s.allowed(args[0]) {
				return toolResult{}, false
			}
			if _, err := s.folderFor(rec); err != nil {
				return toolResult{}, false
			}
			if accepted, _ := rec.Fields["accepted"].(string); accepted != line {
				return toolResult{}, false
			}
			q = question{"Run a program on this computer?",
				press + " It runs " + strings.TrimSpace(line) + " as you, with access to your files. What it does can't be undone from Sameway.",
				"Run it", "Don't run it"}
		case "mqtt":
			topic, _ := rec.Fields["topic"].(string)
			payload, _ := rec.Fields["payload"].(string)
			q = question{"Send a signal to one of your devices?",
				fmt.Sprintf("%s It tells %s: %q. The device may act on it straight away.", press, topic, clip(payload, 120)),
				"Send it", "Don't send"}
		default:
			url, _ := rec.Fields["url"].(string)
			method, _ := rec.Fields["method"].(string)
			body, _ := rec.Fields["body"].(string)
			what := "It contacts " + url + "."
			if body != "" && method != http.MethodGet {
				what = fmt.Sprintf("It sends this to %s: %q.", url, clip(body, 160))
			}
			q = question{"Send something from Sameway to " + host(url) + "?",
				press + " " + what + " Once sent, it can't be unsent.",
				"Send it", "Don't send"}
		}
		return s.ask(q, map[string]any{"tool": "run_action", "id": id}), true
	}
	return toolResult{}, false
}

// ask records the question, in the code's words, with the call a Yes runs.
func (s *Service) ask(q question, action map[string]any) toolResult {
	r := s.propose(q.ask, action)
	if r.isErr || r.change == nil {
		return r
	}
	s.Store.Update(ProposalType, r.change.ID, map[string]any{"detail": q.detail, "yes": q.yes, "no": q.no})
	r.text += ". It was put to them as: " + q.ask + " " + q.detail
	return r
}

// setting is what a setting says now, when the workspace can say.
func (s *Service) setting(key string) string {
	if s.Setting == nil {
		return ""
	}
	return s.Setting(key)
}

// host is the name of the place an address points at, as a person knows
// it: example.com for https://example.com/hooks/1.
func host(addr string) string {
	if u, err := url.Parse(strings.TrimSpace(addr)); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return addr
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "nothing"
	}
	return s
}

// privateAddr says an address only this machine can reach.
func privateAddr(addr string) bool {
	h := addr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		h = addr[:i]
	}
	h = strings.Trim(h, "[]")
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}

// runAgreed is what a person's Yes to a proposal does: the call they were
// asked about, as it was put to them, without asking again.
func (s *Service) runAgreed(call llm.ToolCall) toolResult {
	var args struct {
		ID    string `json:"id"`
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	json.Unmarshal(call.Args, &args)
	switch call.Name {
	case "run_action":
		return s.Run(context.Background(), args.ID, s.current)
	case "set_setting":
		return s.setSetting(args.Key, args.Value)
	case "let_in":
		var a letInArgs
		json.Unmarshal(call.Args, &a)
		return s.letIn(a)
	}
	return s.runTool(call)
}

// keptBySystem refuses fields Sameway keeps itself, such as whether a
// command was accepted: the assistant may read them, never write them.
// Accepting a command is the person's, on the card that asks them.
func keptBySystem(t *schema.Type, fields map[string]any) (toolResult, bool) {
	for _, f := range t.Fields {
		if _, sent := fields[f.Name]; sent && f.ReadOnly {
			return fail("%s is kept by Sameway and cannot be set by the assistant; leave it out", f.Name), true
		}
	}
	return toolResult{}, false
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// quoted is a name as a person reads it quoted.
func quoted(s string) string { return "“" + s + "”" }
