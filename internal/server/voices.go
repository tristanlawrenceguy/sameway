package server

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A call recorded with the computer's sound knows, second by second, who
// was heard: the call (them) or only the microphone (me). The page says
// it as a letter a second (28-speech.js), kept beside the recording as
// its .voices file, the way its transcript is kept as its .vtt; and when
// the recording is written down, each line is said by whoever was heard
// most while it was spoken. Me is the owner's name when a person has
// their address; them is the other person's name when the meeting has
// only one.

var voicesShape = regexp.MustCompile(`^[mt.]+$`)

// mostVoices is the longest a recording can say, a second a letter: a day.
const mostVoices = 86400

// keepVoices puts who was heard beside a recording just added.
func (s *Server) keepVoices(r *http.Request, rec *store.Record) {
	v := r.FormValue("voices")
	if len(v) > mostVoices || !voicesShape.MatchString(v) || !strings.ContainsAny(v, "mt") {
		return
	}
	if path, ok := s.voicesPath(rec); ok {
		os.WriteFile(path, []byte(v), 0o644)
	}
}

func (s *Server) voicesPath(rec *store.Record) (string, bool) {
	path, ok := s.storedPath(rec)
	if !ok {
		return "", false
	}
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".voices", true
}

// namedVoices gives each line that names nobody the one heard most while
// it was spoken, when the recording knows.
func (s *Server) namedVoices(rec *store.Record, cues []convert.Cue) []convert.Cue {
	path, ok := s.voicesPath(rec)
	if !ok {
		return cues
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return cues
	}
	me, them := s.voiceNames(rec)
	for i, c := range cues {
		if c.Speaker != "" {
			continue
		}
		end := c.End
		if end <= c.Start {
			end = c.Start + 5
			if i+1 < len(cues) && cues[i+1].Start > c.Start {
				end = cues[i+1].Start
			}
		}
		m, t := 0, 0
		for sec := int(c.Start); sec < int(end)+1 && sec < len(data); sec++ {
			switch data[sec] {
			case 'm':
				m++
			case 't':
				t++
			}
		}
		switch {
		case t > m:
			cues[i].Speaker = them
		case m > t:
			cues[i].Speaker = me
		}
	}
	return cues
}

// voiceNames are what me and them are called: the owner's name, when a
// person has their address, and the meeting's one other person.
func (s *Server) voiceNames(rec *store.Record) (me, them string) {
	me, them = "Me", "Them"
	pt, ok := s.app.Types.Get("person")
	if !ok {
		return
	}
	ownerID := ""
	if login := s.app.Records.Owner.Login; login != "" {
		if found, _ := query.Filter(s.app.Store, pt, []string{"email=" + login}, "", 1, time.Now()); len(found) == 1 {
			me, ownerID = s.title(pt, found[0]), found[0].ID
		}
	}
	et, ok := s.app.Types.Get(records.EventType)
	if !ok {
		return
	}
	if f, has := et.Field("people"); !has || !f.RefList() {
		return
	}
	meetings, _ := query.Filter(s.app.Store, et, []string{"recording=" + rec.ID}, "", 1, time.Now())
	if len(meetings) != 1 {
		return
	}
	var others []string
	people, _ := meetings[0].Fields["people"].([]any)
	for _, p := range people {
		if id, _ := p.(string); id != "" && id != ownerID {
			others = append(others, id)
		}
	}
	if len(others) == 1 {
		if p, err := s.app.Store.Get("person", others[0]); err == nil {
			them = s.title(pt, p)
		}
	}
	return
}
