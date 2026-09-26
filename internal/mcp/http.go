package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// ServeHTTP is the same server over HTTP, the way a hosted client reaches
// it: one JSON-RPC message or a batch in a POST body, the replies in the
// response body. Streamable HTTP clients that ask to listen with GET are
// told this server only answers; nothing here is pushed.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "send JSON-RPC messages by POST; this server answers and does not stream", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<20))
	if err != nil {
		http.Error(w, "could not read the request", http.StatusBadRequest)
		return
	}
	body = []byte(strings.TrimSpace(string(body)))
	var reqs []request
	batch := len(body) > 0 && body[0] == '['
	if batch {
		err = json.Unmarshal(body, &reqs)
	} else {
		var one request
		err = json.Unmarshal(body, &one)
		reqs = []request{one}
	}
	if err != nil || len(reqs) == 0 {
		writeRPC(w, http.StatusBadRequest, []response{{JSONRPC: "2.0", Error: &rpcError{codeParse, "the body was not JSON-RPC"}}}, false)
		return
	}
	var out []response
	ctx := context.WithValue(r.Context(), offKey{}, offMachine(r))
	for _, req := range reqs {
		result, rpcErr := s.handle(ctx, req)
		if len(req.ID) == 0 || string(req.ID) == "null" {
			continue // a notification wants no answer
		}
		resp := response{JSONRPC: "2.0", ID: req.ID, Result: result}
		if rpcErr != nil {
			resp.Result, resp.Error = nil, rpcErr
		}
		out = append(out, resp)
	}
	if len(out) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeRPC(w, http.StatusOK, out, batch)
}

func writeRPC(w http.ResponseWriter, status int, out []response, batch bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	if batch {
		enc.Encode(out)
		return
	}
	enc.Encode(out[0])
}

// Bearer wraps the HTTP server with the token the workspace names: a
// request without it is refused with a sentence saying what to send.
// Over the tailnet, Tailscale has already said who is asking and the
// workspace has let them in, so no token is asked of them. Otherwise,
// without a token there is no HTTP MCP at all.
func Bearer(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fromTailnet(r) {
			next.ServeHTTP(w, r)
			return
		}
		if token == "" {
			http.Error(w, "MCP over HTTP is off: set the environment variable named by mcp.token_env in workspace.yaml (SAMEWAY_MCP_TOKEN by default) and start sameway again", http.StatusForbidden)
			return
		}
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got != token {
			w.Header().Set("WWW-Authenticate", `Bearer realm="sameway"`)
			http.Error(w, "send the workspace's MCP token as Authorization: Bearer <token>", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
