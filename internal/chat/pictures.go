package chat

import (
	"regexp"
	"strings"
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Pictures come to the model with the conversation, for a model that can
// see: one attached to a message, and one a message is about (its page
// named, or its link pasted), so "what does this say?" or "describe this
// for someone who cannot see it" is answered from the picture itself. The
// three most recent are sent as pictures; older ones are named, not sent
// again every turn. A model that cannot see is told a picture was there.

// picturesSent is how many pictures a turn carries at most.
const picturesSent = 3

var recordID = regexp.MustCompile(`\b[a-z0-9]{16}\b`)

// pictureIDs are the pictures a person's message comes with or is about.
func (s *Service) pictureIDs(content, attached string) []string {
	var ids []string
	seen := map[string]bool{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		if rec, err := s.Store.Get(records.FileType, id); err == nil && rec.Fields["kind"] == "image" {
			ids = append(ids, id)
		}
	}
	add(attached)
	for _, id := range recordID.FindAllString(content, 8) {
		add(id)
	}
	return ids
}

// withPictures puts the pictures on the person's turns, newest first,
// up to picturesSent; each older one is named in words instead.
func (s *Service) withPictures(msgs []llm.Message, ids [][]string) {
	if s.Picture == nil {
		return
	}
	sent := 0
	for i := len(msgs) - 1; i >= 0; i-- {
		for _, id := range ids[i] {
			if sent >= picturesSent {
				msgs[i].Content += "\n[A picture came with this earlier, filed at /t/" + records.FileType + "/" + id + ".]"
				continue
			}
			if img, ok := s.Picture(id); ok {
				msgs[i].Images = append(msgs[i].Images, img)
				sent++
			}
		}
	}
}

// cannotSee says a model refused a turn for its pictures.
func cannotSee(err error) bool {
	e := strings.ToLower(err.Error())
	for _, word := range []string{"image", "vision", "multimodal", "multi-modal"} {
		if strings.Contains(e, word) {
			return true
		}
	}
	return false
}

// withoutPictures is a turn with its pictures taken out, and a word to
// the model that they were there.
func withoutPictures(req llm.Request) (llm.Request, bool) {
	had := false
	out := req
	out.Messages = make([]llm.Message, len(req.Messages))
	for i, m := range req.Messages {
		if len(m.Images) > 0 {
			had = true
			m.Content += "\n[A picture came with this, but the model in use cannot see pictures. Say so plainly, answer from its title and description if it has one, and say that a model that sees pictures (Claude, or a vision model such as llava or qwen2.5-vl on this computer) would read it.]"
			m.Images = nil
		}
		out.Messages[i] = m
	}
	return out, had
}

// What each model has shown of whether it sees pictures, by its name: a
// fact about the model, learned the first time one is sent, so a page can
// say it before the person sends another. It is not kept past a restart.
var sight sync.Map // provider name → bool

// Sees says whether the model has shown it sees pictures, and whether
// that is known yet.
func Sees(model string) (sees, known bool) {
	v, ok := sight.Load(model)
	if !ok {
		return false, false
	}
	return v.(bool), true
}

func hasPictures(req llm.Request) bool {
	for _, m := range req.Messages {
		if len(m.Images) > 0 {
			return true
		}
	}
	return false
}
