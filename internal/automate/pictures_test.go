package automate

import (
	"context"
	"errors"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// fails answers every turn with one error, and counts the turns.
type fails struct {
	err   error
	turns int
}

func (f *fails) Name() string { return "fails" }
func (f *fails) Complete(context.Context, llm.Request) (*llm.Response, error) {
	f.turns++
	return nil, f.err
}

// A model that refuses pictures is asked again without them; one that
// timed out is not, so a model that hangs costs one wait, not two.
func TestOnlyAModelThatRefusesPicturesIsAskedAgain(t *testing.T) {
	t.Parallel()
	pics := []llm.Image{{Type: "image/png", Data: []byte("x")}}
	for _, c := range []struct {
		err   error
		turns int
	}{
		{errors.New("400: this model does not support image input"), 2},
		{context.DeadlineExceeded, 1},
		{errors.New("dial tcp 127.0.0.1:11434: connection refused"), 1},
	} {
		model := &fails{err: c.err}
		s := &Service{Model: func() llm.Provider { return model }}
		var out struct{}
		s.askSeeing(context.Background(), "system", "question", pics, &out)
		if model.turns != c.turns {
			t.Errorf("after %q the model is asked %d times, want %d", c.err, model.turns, c.turns)
		}
	}
}
