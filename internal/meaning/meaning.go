// Package meaning finds records by what they are about, not only by the
// words in them: "the plumber" finds "Boiler repair, call Marek". Each
// record's words are turned into a vector by an embedding model on the
// person's own computer (Ollama), kept beside the record, and a search's
// words are turned the same way; the records nearest in meaning come back.
// Nothing leaves the computer.
//
// Measured with nomic-embed-text on ten everyday notes and ten searches
// for what each was about in other words: the right note came first ten
// times in ten, but its likeness ranged 0.46 to 0.65 while the best wrong
// one reached 0.55, so a fixed bar either misses or misleads. What comes
// back is the nearest few, within a short margin of the nearest, above a
// floor.
package meaning

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Models are the embedding models used when Ollama has one, best first.
var Models = []string{"nomic-embed-text", "embeddinggemma", "mxbai-embed-large"}

// FetchModel is the one fetched when a person asks for search by meaning,
// and FetchWords says it as a person reads it.
const (
	FetchModel = "nomic-embed-text"
	FetchWords = "nomic-embed-text, 274 MB"
)

// Floor, Margin and Most shape what comes back: nothing less alike than
// Floor, nothing further than Margin below the nearest, at most Most.
const (
	Floor  = 0.45
	Margin = 0.06
	Most   = 5
)

// Doc is one record as meaning sees it.
type Doc struct {
	Key  string // type/id
	Text string
}

// Near is a record found by meaning, and how alike it is.
type Near struct {
	Key  string
	Like float64
}

// Keep is where vectors are kept between runs: the store's meta table.
type Keep interface {
	Meta(key string) string
	SetMeta(key, value string)
}

// Index keeps a vector for each record and answers what is near a search.
type Index struct {
	Base  string // Ollama, "http://127.0.0.1:11434"
	Model string
	Keep  Keep

	mu   sync.Mutex
	vecs map[string][]float32
}

// Refresh embeds each doc whose words changed since it was last embedded,
// a batch at a time, and forgets docs that are gone. It says how many it
// embedded.
func (x *Index) Refresh(ctx context.Context, docs []Doc) (int, error) {
	x.mu.Lock()
	if x.vecs == nil {
		x.vecs = map[string][]float32{}
	}
	x.mu.Unlock()
	var todo []Doc
	seen := map[string]bool{}
	for _, d := range docs {
		seen[d.Key] = true
		if x.Keep.Meta("meaning:sum:"+d.Key) == sum(x.Model, d.Text) {
			x.load(d.Key)
			continue
		}
		todo = append(todo, d)
	}
	for i := 0; i < len(todo); i += 32 {
		batch := todo[i:min(i+32, len(todo))]
		var texts []string
		for _, d := range batch {
			texts = append(texts, "search_document: "+d.Text)
		}
		vecs, err := x.embed(ctx, texts)
		if err != nil {
			return i, err
		}
		for j, d := range batch {
			x.Keep.SetMeta("meaning:vec:"+d.Key, pack(vecs[j]))
			x.Keep.SetMeta("meaning:sum:"+d.Key, sum(x.Model, d.Text))
			x.mu.Lock()
			x.vecs[d.Key] = vecs[j]
			x.mu.Unlock()
		}
	}
	x.mu.Lock()
	for k := range x.vecs {
		if !seen[k] {
			delete(x.vecs, k)
		}
	}
	x.mu.Unlock()
	return len(todo), nil
}

func (x *Index) load(key string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if _, ok := x.vecs[key]; ok {
		return
	}
	if v := unpack(x.Keep.Meta("meaning:vec:" + key)); v != nil {
		x.vecs[key] = v
	}
}

// Find is the records nearest a search in meaning.
func (x *Index) Find(ctx context.Context, q string) ([]Near, error) {
	vecs, err := x.embed(ctx, []string{"search_query: " + q})
	if err != nil {
		return nil, err
	}
	qv := vecs[0]
	x.mu.Lock()
	var all []Near
	for k, v := range x.vecs {
		all = append(all, Near{k, cosine(qv, v)})
	}
	x.mu.Unlock()
	sort.Slice(all, func(i, j int) bool { return all[i].Like > all[j].Like })
	var out []Near
	for _, n := range all {
		if n.Like < Floor || n.Like < all[0].Like-Margin || len(out) == Most {
			break
		}
		out = append(out, n)
	}
	return out, nil
}

// Ready says whether there is anything to search by meaning.
func (x *Index) Ready() bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	return len(x.vecs) > 0
}

func (x *Index) embed(ctx context.Context, texts []string) ([][]float32, error) {
	body, _ := json.Marshal(map[string]any{"model": x.Model, "input": texts, "keep_alive": "30m"})
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, x.Base+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var out struct {
		Embeddings [][]float32 `json:"embeddings"`
		Error      string      `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != "" || len(out.Embeddings) != len(texts) {
		return nil, fmt.Errorf("the embedding model answered: %s", out.Error)
	}
	return out.Embeddings, nil
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		if i >= len(b) {
			break
		}
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func sum(model, text string) string {
	h := sha256.Sum256([]byte(model + "\x00" + text))
	return hex.EncodeToString(h[:8])
}

func pack(v []float32) string {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(f))
	}
	return base64.StdEncoding.EncodeToString(b)
}

func unpack(s string) []float32 {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) == 0 || len(b)%4 != 0 {
		return nil
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return v
}
