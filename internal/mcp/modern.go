package mcp

import (
	"context"
	"encoding/json"
)

// MCP 2026-07-28 has no handshake: every request says in _meta which
// revision it speaks and who sends it, and every result says it is
// complete. A request that names a revision is answered that way; one that
// does not is answered as before, so clients that still initialize keep
// working on the same server, over stdio and HTTP alike.

// modernProtocol is the revision a request names in _meta.
const modernProtocol = "2026-07-28"

const (
	metaServer = "io.modelcontextprotocol/serverInfo"
	// codeUnsupportedVersion is the spec's UnsupportedProtocolVersionError.
	codeUnsupportedVersion = -32022
)

// meta is what a modern request carries in params._meta.
type meta struct {
	Version string          `json:"io.modelcontextprotocol/protocolVersion"`
	Client  json.RawMessage `json:"io.modelcontextprotocol/clientInfo"`
}

func metaOf(params json.RawMessage) meta {
	var p struct {
		Meta meta `json:"_meta"`
	}
	json.Unmarshal(params, &p)
	return p.Meta
}

// modern answers a request that names its revision: refused with the
// revisions this server has when it names another, otherwise answered as
// any request is, its client named by what it says it is, and its result
// marked complete with who answered.
func (s *Server) modern(ctx context.Context, req request, m meta) (any, *rpcError) {
	if m.Version != modernProtocol {
		return nil, &rpcError{codeUnsupportedVersion, "Unsupported protocol version",
			map[string]any{"supported": []string{modernProtocol}, "requested": m.Version}}
	}
	var info clientInfo
	if json.Unmarshal(m.Client, &info) == nil && (info.Name != "" || info.Title != "") {
		connOf(ctx).call(info) // agent.go
	}
	result, rpcErr := s.answer(ctx, req)
	if r, ok := result.(map[string]any); ok {
		r["resultType"] = "complete"
		r["_meta"] = map[string]any{metaServer: s.info()}
		if req.Method == "tools/list" {
			// The list is the same order every time (the registry's), so a
			// client may keep it; but types and permissions change it, so
			// only for this client and not for long.
			r["ttlMs"], r["cacheScope"] = 0, "private"
		}
	}
	return result, rpcErr
}

func (s *Server) info() map[string]any {
	return map[string]any{"name": "sameway", "version": s.Version}
}

// capabilities are what the server offers, on initialize and on discover.
// The list changes for the assistant (turn_tools.go): Claude Code reads it
// again on the notification only when told so.
func (s *Server) capabilities() map[string]any {
	return map[string]any{"tools": map[string]any{"listChanged": s.Only != nil}}
}

// discover is server/discover: the revisions, what it offers, who it is.
func (s *Server) discover() map[string]any {
	return map[string]any{
		"resultType":        "complete",
		"supportedVersions": []string{modernProtocol},
		"capabilities":      s.capabilities(),
		"instructions":      instructions,
		"_meta":             map[string]any{metaServer: s.info()},
		"ttlMs":             0,
		"cacheScope":        "private",
	}
}
