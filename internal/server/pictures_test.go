package server_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// sees records what it is sent, and can refuse pictures as a model that
// cannot see does.
type sees struct {
	blind bool
	seen  []llm.Request
}

func (s *sees) Name() string { return "sees" }
func (s *sees) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	s.seen = append(s.seen, req)
	for _, m := range req.Messages {
		if s.blind && len(m.Images) > 0 {
			return nil, errors.New(`400: this model does not support image input`)
		}
	}
	return &llm.Response{Text: "A fern on a shelf."}, nil
}

func pngOf(t *testing.T, w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

// A picture attached to a message, or named in one, is shown to a model
// that can see: small ones as they are, large ones made no longer than
// 1568 pixels; only the three newest go each turn; a model that cannot
// see answers again, told a picture was there.
func TestPicturesAreShownToTheAssistant(t *testing.T) {
	a, h := newApp(t)
	model := &sees{}
	a.Chat.Provider = model
	upload := func(name string, data []byte) string {
		body, ct := multipartFile(t, name, string(data), nil)
		return strings.TrimPrefix(do(t, h, http.MethodPost, "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	}
	small := upload("Fern.png", pngOf(t, 40, 30))
	big := upload("Wall.png", pngOf(t, 3000, 2000))

	if _, err := a.Chat.SendFile(context.Background(), "", "What is this?", small); err != nil {
		t.Fatal(err)
	}
	last := model.seen[len(model.seen)-1].Messages
	img := last[len(last)-1].Images
	if len(img) != 1 || img[0].Type != "image/png" {
		t.Fatalf("the attached picture goes as it is: %+v", img)
	}

	if _, err := a.Chat.Send(context.Background(), "Describe "+big+" for someone who cannot see it"); err != nil {
		t.Fatal(err)
	}
	last = model.seen[len(model.seen)-1].Messages
	img = last[len(last)-1].Images
	if len(img) != 1 || img[0].Type != "image/jpeg" {
		t.Fatalf("a picture named by its id goes too, made smaller: %+v", img)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(img[0].Data))
	if err != nil || cfg.Width != 1568 || cfg.Height != 1045 {
		t.Errorf("no longer than 1568 pixels, in proportion: %v %+v", err, cfg)
	}

	for i := 0; i < 3; i++ {
		a.Chat.SendFile(context.Background(), "", "And this?", small)
	}
	count := 0
	for _, m := range model.seen[len(model.seen)-1].Messages {
		count += len(m.Images)
	}
	if count != 3 {
		t.Errorf("only the three newest pictures go each turn, got %d", count)
	}

	model.blind = true
	rec, err := a.Chat.SendFile(context.Background(), "", "What does it say?", small)
	if err != nil {
		t.Fatalf("a model that cannot see still answers: %v", err)
	}
	final := model.seen[len(model.seen)-1].Messages
	if n := final[len(final)-1]; len(n.Images) != 0 || !strings.Contains(n.Content, "cannot see pictures") {
		t.Errorf("it is told a picture was there: %+v", n)
	}
	if content, _ := rec.Fields["content"].(string); content == "" {
		t.Error("and its answer is kept")
	}
}

// A picture with no description leads to asking the assistant for one,
// the picture named so it goes with the question.
func TestAPictureWithNoDescriptionLeadsToADraft(t *testing.T) {
	_, h := newApp(t)
	body, ct := multipartFile(t, "Fern.png", string(pngOf(t, 20, 20)), nil)
	id := strings.TrimPrefix(do(t, h, http.MethodPost, "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	page := get(t, h, "/t/file/"+id).Body.String()
	if !strings.Contains(page, "Ask the assistant to describe it") || !strings.Contains(page, "%2Ft%2Ffile%2F"+id) {
		t.Errorf("the page offers a draft description:\n%.2000s", page)
	}
}
