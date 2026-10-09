package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Telling speakers apart (speech/speakers.go): once the owner has it, a
// recording written down whole has each line said by Speaker 1, Speaker 2
// and so on, in the order they are first heard, as many as its meeting's
// people when it has them; a person gives each a name by editing the
// text. A call recorded with the computer's sound says me and them
// instead, which is surer (voices.go), and a recording written down in
// parts is left as it is, since a voice cannot be followed from one part
// to the next.

// speakersKit is how speakers are told apart here, or nil.
type speakersKit struct {
	Ready   func() bool
	Install func(ctx context.Context, p speech.Progress) error
	Diarize func(ctx context.Context, wav string, speakers int) ([]speech.Turn, error)
}

func (s *Server) speakers() *speakersKit {
	s.speech.mu.Lock()
	defer s.speech.mu.Unlock()
	if s.speech.speakers == nil {
		dir, _ := speech.Dir()
		s.speech.speakers = &speakersKit{
			Ready:   func() bool { return dir != "" && speech.SpeakersReady(dir) },
			Install: func(ctx context.Context, p speech.Progress) error { return speech.InstallSpeakers(ctx, nil, dir, p) },
			Diarize: func(ctx context.Context, wav string, n int) ([]speech.Turn, error) {
				return speech.Diarize(ctx, dir, wav, n)
			},
		}
	}
	return s.speech.speakers
}

// UseSpeakers sets how speakers are told apart, for tests.
func (s *Server) UseSpeakers(ready func() bool, diarize func(ctx context.Context, wav string, n int) ([]speech.Turn, error)) {
	s.speech.mu.Lock()
	s.speech.speakers = &speakersKit{Ready: ready, Diarize: diarize, Install: func(context.Context, speech.Progress) error { return nil }}
	s.speech.mu.Unlock()
}

// whoSpoke names each line of a recording written down whole, when
// speakers can be told apart here and nothing better says who spoke.
func (s *Server) whoSpoke(rec *store.Record, wav string, start float64, cues []convert.Cue) []convert.Cue {
	kit := s.speakers()
	if kit == nil || kit.Ready == nil || !kit.Ready() || len(cues) == 0 {
		return cues
	}
	if path, ok := s.voicesPath(rec); ok {
		if _, err := os.Stat(path); err == nil {
			return cues // me and them, from the call itself
		}
	}
	for _, c := range cues {
		if c.Speaker != "" {
			return cues
		}
	}
	turns, err := kit.Diarize(context.Background(), wav, s.howManySpeak(rec))
	if err != nil || len(turns) == 0 {
		return cues
	}
	voices := map[int]bool{}
	for _, t := range turns {
		voices[t.Speaker] = true
	}
	if len(voices) < 2 {
		return cues // one voice says nothing worth saying
	}
	for i, c := range cues {
		best, most := 0, 0.0
		for _, t := range turns {
			over := min(c.End, t.End+start) - max(c.Start, t.Start+start)
			if over > most {
				best, most = t.Speaker, over
			}
		}
		if best > 0 {
			cues[i].Speaker = fmt.Sprintf("Speaker %d", best)
		}
	}
	return cues
}

// howManySpeak is how many people are in the recording's meeting, the
// owner among them, or 0 when nobody says.
func (s *Server) howManySpeak(rec *store.Record) int {
	et, ok := s.app.Types.Get(records.EventType)
	if !ok {
		return 0
	}
	if f, has := et.Field("people"); !has || !f.RefList() {
		return 0
	}
	meetings, _ := query.Filter(s.app.Store, et, []string{"recording=" + rec.ID}, "", 1, s.now())
	if len(meetings) != 1 {
		return 0
	}
	people, _ := meetings[0].Fields["people"].([]any)
	if len(people) == 0 {
		return 0
	}
	return len(people) + 1 // they, and whoever is recording
}

// speakersGet is the owner's press that gets telling speakers apart.
func (s *Server) speakersGet(w http.ResponseWriter, r *http.Request) {
	kit := s.speakers()
	if kit.Ready() {
		s.tell(w, r, outcome{Title: "Telling speakers apart is already on this computer"}, "/help")
		return
	}
	if !s.speechKit().Ready() {
		s.tell(w, r, outcome{Failed: true, Title: "Speech-to-text first", Text: "Telling speakers apart works on what speech-to-text writes down, so it needs that on this computer first."}, "/help")
		return
	}
	s.record(r, records.Change{Action: "started", Detail: "getting speaker separation for this computer (" + sizeWords(speech.SpeakersSize()) + ")"})
	go func() {
		err := kit.Install(context.Background(), nil)
		if err != nil {
			records.Record(s.app.Store, "system", records.Change{Action: "failed", Detail: "getting speaker separation: " + err.Error()})
		} else {
			records.Record(s.app.Store, "system", records.Change{Action: "added", Detail: "speaker separation for this computer"})
		}
		s.Changed()
	}()
	s.tell(w, r, outcome{Title: "Getting it", Text: "It downloads once, about " + sizeWords(speech.SpeakersSize()) + ". Recordings written down after it is here say who spoke."}, "/help")
}
