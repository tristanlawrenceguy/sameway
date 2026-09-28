package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Two people can change the same text at once, on two computers: each
// writes over what they saw, not knowing of the other. The latest still
// wins, so every copy agrees; but the other version is not thrown away.
// Each stamp of a text field says what it was written over (its base, a
// hash of the value before). When a stamp arrives whose base is not what
// this copy holds, and what this copy holds was not written over it
// either, neither writer saw the other: the one that lost is kept as a
// clash, which the record's page offers back. Both copies see the same
// clash and name it the same, so it is one clash, kept the same.

// ClashType holds the versions that lost to another written at the same
// time.
const ClashType = "clash"

func (s *Store) migrateClash() {
	s.db.Exec(`ALTER TABLE _state ADD COLUMN base TEXT`) // already there after the first time
}

func hashRaw(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:6])
}

// clashes says whether a field is text a person writes at length, the kind
// two people might both be in the middle of.
func clashes(f *schema.Field) bool {
	return f != nil && (f.Type == "text" || f.Type == "markdown")
}

// baseOf is what a local write of a text field was written over.
func (s *Store) baseOf(t *schema.Type, field string, before map[string]any) string {
	f, _ := t.Field(field)
	if before == nil || !clashes(f) {
		return ""
	}
	raw, _ := json.Marshal(before[field])
	return hashRaw(string(raw))
}

// clash keeps the losing version when a stamp that arrived and what this
// copy holds were each written without seeing the other.
func (s *Store) clash(st Stamp, have, cur, curBase string) {
	if have == "" || cur == string(st.Value) || st.Base == "" {
		return
	}
	t, ok := s.types.Get(st.Type)
	if !ok {
		return
	}
	if f, _ := t.Field(st.Field); !clashes(f) {
		return
	}
	if st.Base == hashRaw(cur) || curBase == hashRaw(string(st.Value)) {
		return // one was written over the other: an ordinary edit
	}
	lost, clock := cur, have
	if st.Clock < have {
		lost, clock = string(st.Value), st.Clock
	}
	if _, ok := s.types.Get(ClashType); !ok {
		return
	}
	id := "c" + hashRaw(st.Type+"|"+st.ID+"|"+st.Field+"|"+clock)
	if _, err := s.Get(ClashType, id); err == nil {
		return
	}
	var text string
	json.Unmarshal([]byte(lost), &text)
	now := time.Now().UTC()
	s.Put(ClashType, id, map[string]any{"target": st.Type, "target_id": st.ID, "field": st.Field, "text": text, "origin": originOf(clock), "state": "open"}, now, now)
}
