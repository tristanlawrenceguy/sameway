package llm

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"
)

// With SAMEWAY_CLAUDE_CODE_MODEL set (haiku will do), Claude Code is given
// a real picture the way a turn gives it one, and says what it shows: the
// folder it may read is enough for it to look.
func TestClaudeCodeSeesAPicture(t *testing.T) {
	t.Parallel()
	model := os.Getenv("SAMEWAY_CLAUDE_CODE_MODEL")
	if model == "" {
		t.Skip("set SAMEWAY_CLAUDE_CODE_MODEL to run one real turn through Claude Code")
	}
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			c := color.RGBA{255, 255, 255, 255}
			if x > 50 && x < 150 && y > 50 && y < 150 {
				c = color.RGBA{220, 20, 20, 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	c := Presets["claude-code"]
	c.Model, c.Workspace, c.Timeout = model, t.TempDir(), 3*time.Minute
	resp, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "What colour is the square in the picture? Answer with one word.", Images: []Image{{Type: "image/png", Data: b.Bytes()}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(resp.Text), "red") {
		t.Errorf("Claude Code should see a red square, said %q", resp.Text)
	}
	entries, _ := os.ReadDir(c.Workspace + "/.sameway-pictures")
	if len(entries) != 0 {
		t.Error("the pictures go with the turn")
	}
}
