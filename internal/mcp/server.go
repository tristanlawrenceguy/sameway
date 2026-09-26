// Package mcp serves a workspace to any Model Context Protocol client over
// stdio: newline-delimited JSON-RPC 2.0, the way Claude Code, Claude Desktop
// and the other MCP hosts start a server.
//
// The tools are the chat assistant's own list, plus reading. An agent in an
// MCP host drives sameway with the same verbs the assistant has, under the
// same checks, into the same activity log; and a tool added to the chat
// service is on MCP the same moment, with nothing written twice.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// protocolVersion is the MCP revision this server speaks.
const protocolVersion = "2025-06-18"

// Server answers one client for as long as its input lasts.
type Server struct {
	App     *app.App
	Version string
	In      io.Reader
	Out     io.Writer

	mu sync.Mutex
	// http is the web server over the same app, for tools that read a
	// page the way the API does.
	http http.Handler
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSON-RPC error codes the protocol reserves.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// Serve reads requests until the input ends or the context is cancelled.
// A notification (no id) gets no reply; everything else gets exactly one.
func (s *Server) Serve(ctx context.Context) error {
	sc := bufio.NewScanner(s.In)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			s.write(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParse, "the line was not valid JSON: " + err.Error()}})
			continue
		}
		notification := len(req.ID) == 0 || string(req.ID) == "null"
		result, rpcErr := s.handle(ctx, req)
		if notification {
			continue
		}
		s.write(response{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr})
	}
	return sc.Err()
}

func (s *Server) write(resp response) {
	raw, err := json.Marshal(resp)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Out.Write(append(raw, '\n'))
}

// handle answers one request. Unknown methods are the protocol's own error,
// so a host probing for capabilities this server lacks gets a clean no.
func (s *Server) handle(ctx context.Context, req request) (any, *rpcError) {
	if req.JSONRPC != "2.0" {
		return nil, &rpcError{codeInvalidRequest, `jsonrpc must be "2.0"`}
	}
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "sameway", "version": s.Version},
			"instructions": "This is a Sameway workspace: content records of the types the workspace declares, " +
				"and a canvas of components. Call describe first for the types, their fields, the components " +
				"and every surface; then find_records, get_record, create_record and update_record for content, " +
				"and the canvas tools for what the person sees.",
		}, nil
	case "notifications/initialized", "notifications/cancelled":
		return nil, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": s.listFor(ctx)}, nil
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
			return nil, &rpcError{codeInvalidParams, "tools/call needs params.name and params.arguments"}
		}
		text, isError := refusedOff, true
		if ok, svc := s.may(ctx, params.Name); ok {
			text, isError = s.call(ctx, svc, params.Name, params.Arguments)
		}
		return map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
			"isError": isError,
		}, nil
	}
	return nil, &rpcError{codeMethodNotFound, fmt.Sprintf("method %q is not one this server has; it offers initialize, ping, tools/list and tools/call", req.Method)}
}
