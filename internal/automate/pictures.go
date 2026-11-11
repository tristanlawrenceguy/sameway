package automate

import (
	"context"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A photo of a letter, a bill or a school notice, shared from a phone, is
// a note that links to the picture, and its words are only the file's
// name: tagged and suggested from that, it was "nothing to do". So the
// pictures a record links to go with its question, for a model that can
// see; a model that cannot is asked again without them.

const picturesSeen = 3

var fileLink = regexp.MustCompile(`/t/file/([a-z0-9]{16})`)

// picturesOf are the pictures a record's words link to, up to three.
func (s *Service) picturesOf(rec *store.Record) []llm.Image {
	if s.Pictures == nil {
		return nil
	}
	var out []llm.Image
	for _, id := range linkedFiles(rec) { // pictures_read.go
		if f, err := s.Store.Get(records.FileType, id); err == nil && strings.TrimSpace(str(f.Fields["text"])) != "" {
			continue // read already: its words are the record's (record_words.go)
		}
		out = append(out, s.Pictures(id)...)
		if len(out) >= picturesSeen {
			return out[:picturesSeen] // a scan's first pages; its words are read in full once (pictures_read.go)
		}
	}
	return out
}

// askSeeing is askJSON with the record's pictures beside the question.
func (s *Service) askSeeing(ctx context.Context, system, question string, pics []llm.Image, out any) error {
	if len(pics) == 0 {
		return s.askJSON(ctx, system, question, out)
	}
	err := s.ask(ctx, system, question+"\n\nThe pictures the record links to come with it: read what they say as part of the record.", pics, out)
	if err != nil {
		err = s.askJSON(ctx, system, question, out) // a model that cannot see
	}
	return err
}
