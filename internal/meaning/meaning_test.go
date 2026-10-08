package meaning

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type keep struct {
	sync.Mutex
	m map[string]string
}

func (k *keep) Meta(key string) string { k.Lock(); defer k.Unlock(); return k.m[key] }
func (k *keep) SetMeta(key, v string)  { k.Lock(); k.m[key] = v; k.Unlock() }

// Records are read again only when their words change, and a search finds
// the nearest few, within a margin of the nearest, above a floor.
func TestTheNearestInMeaningAreFound(t *testing.T) {
	embedded := 0
	// A stand-in model: each text is a vector of which topic words it has.
	topics := []string{"plumb boiler pipe", "passport travel", "flour egg pancake"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Input []string }
		json.NewDecoder(r.Body).Decode(&in)
		var out [][]float32
		for _, text := range in.Input {
			if strings.HasPrefix(text, "search_document: ") {
				embedded++
			}
			v := make([]float32, len(topics))
			for i, words := range topics {
				for _, w := range strings.Fields(words) {
					if strings.Contains(strings.ToLower(text), w) {
						v[i] += 1
					}
				}
			}
			v = append(v, 0.2) // something shared, as real vectors have
			out = append(out, v)
		}
		json.NewEncoder(w).Encode(map[string]any{"embeddings": out})
	}))
	defer srv.Close()
	x := &Index{Base: srv.URL, Model: "m", Keep: &keep{m: map[string]string{}}}
	docs := []Doc{{"note/a", "Boiler repair, call Marek"}, {"note/b", "Renew the passport"}, {"note/c", "Pancakes: flour, egg, milk"}}
	if n, err := x.Refresh(context.Background(), docs); err != nil || n != 3 {
		t.Fatalf("all read the first time: %d %v", n, err)
	}
	docs[1].Text = "Renew the passport before travel"
	if n, _ := x.Refresh(context.Background(), docs); n != 1 {
		t.Errorf("only what changed is read again: %d", n)
	}
	found, err := x.Find(context.Background(), "the plumber and the pipe")
	if err != nil || len(found) == 0 || found[0].Key != "note/a" {
		t.Fatalf("the plumber finds the boiler: %v %v", found, err)
	}
	for _, f := range found {
		if f.Key == "note/c" {
			t.Errorf("not the pancakes: %v", found)
		}
	}
}
