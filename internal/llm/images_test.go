package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var png1 = Image{Type: "image/png", Data: []byte("\x89PNG not really")}

// A picture goes to an OpenAI-compatible server (OpenAI, Ollama, LM
// Studio) as an image part of the person's turn, with the words after it.
func TestAPictureGoesAsAnImagePart(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &body)
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"A fern."},"finish_reason":"stop"}]}`)
	}))
	defer srv.Close()
	o := &OpenAI{BaseURL: srv.URL, Model: "llava"}
	resp, err := o.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "What is this?", Images: []Image{png1}}}})
	if err != nil || resp.Text != "A fern." {
		t.Fatalf("%v %+v", err, resp)
	}
	msg := body["messages"].([]any)[0].(map[string]any)
	parts, ok := msg["content"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("the content is parts: %v", msg)
	}
	img := parts[0].(map[string]any)
	if img["type"] != "image_url" || !strings.HasPrefix(img["image_url"].(map[string]any)["url"].(string), "data:image/png;base64,") || parts[1].(map[string]any)["text"] != "What is this?" {
		t.Errorf("a picture, then the words: %v", parts)
	}
	// With no picture, the content stays a plain string.
	if m := toOpenAI(Message{Role: RoleUser, Content: "Hello"}); m[0].Content != "Hello" {
		t.Errorf("words alone are a string: %#v", m[0].Content)
	}
}

// A picture goes to Anthropic as an image block before the words.
func TestAPictureGoesToAnthropicAsAnImageBlock(t *testing.T) {
	raw, _ := json.Marshal(toAnthropic(Message{Role: RoleUser, Content: "What is this?", Images: []Image{png1}}))
	s := string(raw)
	if !strings.Contains(s, `"type":"image"`) || !strings.Contains(s, `"media_type":"image/png"`) || strings.Index(s, `"type":"image"`) > strings.Index(s, "What is this?") {
		t.Errorf("an image block, then the words: %s", s)
	}
}

// Claude Code reads the conversation as words: each picture is written in
// the workspace, named in its turn, and gone with the turn.
func TestClaudeCodeIsGivenPicturesAsFiles(t *testing.T) {
	ws := t.TempDir()
	c := &Command{Workspace: ws}
	req, rel, err := c.pictures(Request{Messages: []Message{{Role: RoleUser, Content: "Read this", Images: []Image{png1}}}})
	if err != nil || rel == "" {
		t.Fatal(err)
	}
	m := req.Messages[0]
	if len(m.Images) != 0 || !strings.Contains(m.Content, "[A picture came with this, at .sameway-pictures/") || !strings.Contains(m.Content, "/1.png: open it with Read to see it.]") {
		t.Errorf("the turn names its picture: %q", m.Content)
	}
	if b, err := os.ReadFile(filepath.Join(ws, rel, "1.png")); err != nil || string(b) != string(png1.Data) {
		t.Errorf("and the picture is there: %v", err)
	}
	if _, rel2, _ := c.pictures(Request{Messages: []Message{{Role: RoleUser, Content: "No picture"}}}); rel2 != "" {
		t.Error("a turn with no picture writes nothing")
	}
}
