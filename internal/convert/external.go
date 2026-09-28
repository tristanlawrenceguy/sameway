package convert

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// A converter is what the workspace names for a format the built-in
// readers cannot do justice to: a URL, such as docling's server, or a
// command line with {file} where the file's path goes. Either answers with
// Markdown: a command on its standard output, a URL as text or as JSON
// with a markdown field. The workspace file names converters by
// extension, so a person chooses docling for PDFs and pandoc for Word if
// they like, and the built-in reader for the rest.

// Timeout bounds one conversion; a scanned PDF can take a while.
var Timeout = 5 * time.Minute

// HTTPClient posts files to URL converters. Tests point it at a local server.
var HTTPClient = &http.Client{Timeout: Timeout}

// External converts a file with the converter named for its extension.
func External(ctx context.Context, converter, name, path string) (string, error) {
	converter = strings.TrimSpace(converter)
	if converter == "" {
		return "", errors.New("no converter")
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	if strings.HasPrefix(converter, "http://") || strings.HasPrefix(converter, "https://") {
		return post(ctx, converter, name, path)
	}
	return run(ctx, converter, path)
}

func run(ctx context.Context, line, path string) (string, error) {
	if !strings.Contains(line, "{file}") {
		return "", errors.New("a command converter needs {file} where the file's path goes")
	}
	line = strings.ReplaceAll(line, "{file}", quote(path))
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", line)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", line)
	}
	cmd.Dir = filepath.Dir(path)
	var out, errs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errs
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("converter failed: %v: %s", err, strings.TrimSpace(errs.String()))
	}
	md := strings.TrimSpace(out.String())
	if md == "" {
		return "", errors.New("the converter printed nothing")
	}
	return md, nil
}

func quote(p string) string {
	if strings.ContainsAny(p, " \t") {
		return `"` + p + `"`
	}
	return p
}

// post sends the file as a multipart form, the way docling's server and
// most converters take one, and reads Markdown back from text or JSON.
func post(ctx context.Context, url, name, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for _, field := range []string{"files", "file"} {
		part, err := w.CreateFormFile(field, name)
		if err != nil {
			return "", err
		}
		part.Write(data)
	}
	w.WriteField("to_formats", "md")
	w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json, text/markdown, text/plain")
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("converter answered %d: %.200s", resp.StatusCode, raw)
	}
	if md := markdownIn(raw); md != "" {
		return md, nil
	}
	return "", errors.New("the converter answered with no markdown")
}

// markdownIn finds the Markdown in a converter's answer: the body itself
// when it is text, or a markdown-ish field anywhere in JSON.
func markdownIn(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return string(trimmed)
	}
	var v any
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return string(trimmed)
	}
	return find(v)
}

func find(v any) string {
	switch x := v.(type) {
	case map[string]any:
		for _, key := range []string{"md_content", "markdown", "md", "text_content", "content", "text"} {
			if s, ok := x[key].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
		for _, child := range x {
			if s := find(child); s != "" {
				return s
			}
		}
	case []any:
		for _, child := range x {
			if s := find(child); s != "" {
				return s
			}
		}
	}
	return ""
}
