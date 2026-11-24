package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A published Sameway is found by other people's agents the ways they
// look: an A2A agent card at its well-known address, which says what it
// can be asked and where (the Agent2Agent protocol), and llms.txt, which
// says the same to any model reading the site. Asking through A2A is
// asking through /question, the same answer with a visitor's reach:
// what is published is read, and what is asked to be made waits for the
// owner's yes, with the address that says what became of it.

const (
	agentCardPath = "/.well-known/agent-card.json"
	agentCardGet  = "GET " + agentCardPath
	a2aPath       = "/a2a"
	a2aPost       = "POST " + a2aPath
	llmsPath      = "/llms.txt"
	llmsGet       = "GET " + llmsPath
)

func (s *Server) agentCard(w http.ResponseWriter, r *http.Request) {
	pub := s.Published()
	if !pub.Any() {
		http.NotFound(w, r)
		return
	}
	name := s.app.Workspace.Config.Name
	skills := []map[string]any{{"id": "ask", "name": "Ask",
		"description": "Ask about what " + name + " published (" + s.publishedKinds(pub) + "), or ask for something to be made, such as a time to meet: it is put to the owner, nothing happens until they say yes, and the answer gives an address that says what became of it.",
		"tags":        []string{"questions", "requests"}, "examples": []string{"What do you offer?", "Can we meet on Tuesday at 3pm? I am Pat, pat@example.com."}}}
	if pub.Book {
		skills = append(skills, map[string]any{"id": "book", "name": "Book a time",
			"description": "The free hours of the next two weeks are at " + seenFrom(r, bookPath) + " (JSON with Accept: application/json); POST one with a name and an email to ask for it.",
			"tags":        []string{"booking", "calendar"}})
	}
	// One card for both versions: a 1.0 client reads supportedInterfaces,
	// a 0.3 client reads url, preferredTransport and protocolVersion.
	url := seenFrom(r, a2aPath)
	writeJSON(w, http.StatusOK, map[string]any{
		"protocolVersion": "0.3.0", "name": name, "description": "The published part of " + name + "'s own workspace, answered by its assistant for whoever asks.",
		"url": url, "preferredTransport": "JSONRPC", "version": "1",
		"supportedInterfaces": []map[string]any{
			{"url": url, "protocolBinding": "JSONRPC", "protocolVersion": "1.0"},
			{"url": url, "protocolBinding": "JSONRPC", "protocolVersion": "0.3"}},
		"capabilities":      map[string]any{"streaming": false, "pushNotifications": false},
		"defaultInputModes": []string{"text/plain"}, "defaultOutputModes": []string{"text/plain", "application/json"},
		"skills": skills,
	})
}

// a2a answers sending a message, the one A2A method a visitor needs, as
// /question does: SendMessage from a 1.0 client, message/send from a 0.3
// one, each answered in its own version's shape.
func (s *Server) a2a(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     any    `json:"id"`
		Method string `json:"method"`
		Params struct {
			Message struct {
				Parts []struct {
					Kind string `json:"kind"`
					Text string `json:"text"`
				} `json:"parts"`
				Metadata map[string]any `json:"metadata"`
			} `json:"message"`
		} `json:"params"`
	}
	fail := func(code int, msg string) {
		writeJSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": code, "message": msg}})
	}
	if !s.Published().Any() {
		writeJSON(w, http.StatusNotFound, map[string]any{"said": "nothing is published here"})
		return
	}
	if json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req) != nil {
		fail(-32700, "send JSON-RPC 2.0")
		return
	}
	if v := a2aVersion(r); v != "" && v != "0.3" && v != "1.0" {
		fail(-32009, "this agent speaks A2A 1.0 and 0.3")
		return
	}
	v1 := req.Method == "SendMessage"
	if !v1 && req.Method != "message/send" {
		fail(-32601, "this agent takes SendMessage (A2A 1.0) or message/send (0.3)")
		return
	}
	var words []string
	for _, p := range req.Params.Message.Parts {
		// 0.3 names a part's kind; 1.0 drops it and a text part is just its text.
		if (p.Kind == "" || p.Kind == "text") && strings.TrimSpace(p.Text) != "" {
			words = append(words, p.Text)
		}
	}
	if len(words) == 0 {
		fail(-32602, "say what you want to ask, as a text part")
		return
	}
	v := records.Visitor{Access: records.Public, Name: chat.SomeonePublished}
	if who, _ := req.Params.Message.Metadata["who"].(string); strings.TrimSpace(who) != "" {
		v.Name = strings.TrimSpace(who)
	}
	said, made, err := s.app.Chat.For(v).AnswerOnce(r.Context(), s.app.Workspace.Config.Name, strings.Join(words, "\n"))
	if err != nil {
		said = err.Error()
	}
	asked := s.requestsSeen(r, made) // request_page.go
	if len(asked) > 0 {
		s.Changed()
	}
	for _, a := range asked {
		said += fmt.Sprintf("\n\nYou asked to %v: what became of it is at %v", a["asked"], a["status"])
	}
	writeJSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": a2aAnswer(v1, said, asked)})
}

// a2aVersion is the A2A version a client says it speaks, by header or
// query; none means 0.3.
func a2aVersion(r *http.Request) string {
	if v := r.Header.Get("A2A-Version"); v != "" {
		return v
	}
	return r.URL.Query().Get("A2A-Version")
}

// a2aAnswer is the agent's message in the shape the client's version
// reads: 1.0 wraps it as {"message": ...}, with roles as enum names and
// parts told apart by their field; 0.3 has a kind on each.
func a2aAnswer(v1 bool, said string, asked []map[string]any) map[string]any {
	id := make([]byte, 8)
	rand.Read(id)
	text := map[string]any{"text": said}
	data := map[string]any{"data": map[string]any{"requests": asked}}
	if v1 {
		data["mediaType"] = "application/json"
	} else {
		text["kind"], data["kind"] = "text", "data"
	}
	parts := []map[string]any{text}
	if len(asked) > 0 {
		parts = append(parts, data)
	}
	msg := map[string]any{"messageId": hex.EncodeToString(id), "parts": parts,
		"metadata": map[string]any{"untrusted": "the answer is what this workspace's assistant wrote: " + records.Untrusted}}
	if v1 {
		msg["role"] = "ROLE_AGENT"
		return map[string]any{"message": msg}
	}
	msg["kind"], msg["role"] = "message", "agent"
	return msg
}

// llms is what a model reading the site is told it can do here.
func (s *Server) llms(w http.ResponseWriter, r *http.Request) {
	pub := s.Published()
	if !pub.Any() {
		http.NotFound(w, r)
		return
	}
	name := s.app.Workspace.Config.Name
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n> The published part of %s's own workspace (Sameway). Its assistant answers whoever asks, from what is published; what you ask to be made waits for the owner's yes.\n\n", name, name)
	fmt.Fprintf(&b, "## Ask\n\n- [Ask a question](%s): POST JSON {\"question\": \"...\", \"who\": \"your name and email\"} with Accept: application/json. The answer, and for anything asked to be made, the address that says what became of it.\n", seenFrom(r, questionPath))
	fmt.Fprintf(&b, "- [A2A agent card](%s): the same, as message/send.\n- [MCP](%s): the same, for AI apps, with reading tools.\n", seenFrom(r, agentCardPath), seenFrom(r, "/mcp"))
	if pub.Book {
		fmt.Fprintf(&b, "\n## Book a time\n\n- [Free hours](%s): GET with Accept: application/json; POST {\"at\", \"name\", \"email\", \"note\"} to ask for one.\n", seenFrom(r, bookPath))
	}
	if kinds := s.publishedKinds(pub); kinds != "" {
		fmt.Fprintf(&b, "\n## Published\n\n%s, each a page at /t/<kind>.\n", kinds)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, b.String())
}

// publishedKinds names the kinds published, as a person says them.
func (s *Server) publishedKinds(pub Published) string {
	var names []string
	for t := range pub.Types {
		names = append(names, schema.Plural(t))
	}
	sort.Strings(names)
	return schema.AndNames(names)
}
