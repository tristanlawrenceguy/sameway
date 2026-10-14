package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Several records in one request. An agent putting a week's runs or a
// brain dump of tasks into the workspace sent one request a record, each a
// change of its own in the log for the person to undo one at a time; one
// building a body of 106 items by hand lost it to a character it got
// wrong, half sent. POST /api/{type} takes a list as well as one record,
// and PATCH /api/{type} the same change to several ids: all are written or
// none, as one change in the log, which one Undo takes back. Blocks are
// still written one at a time, each checked as it is.

// batchRead is the body when it is a list of records, nil when it is one
// record; the body is put back for the one-record way either way.
func batchRead(r *http.Request) ([]map[string]any, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if t := bytes.TrimLeft(raw, " \t\r\n"); len(t) == 0 || t[0] != '[' {
		return nil, nil
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, errors.New("a list of records is a JSON array of objects of fields: " + web.JSONTrouble(err))
	}
	if len(list) == 0 {
		return nil, errors.New("the list is empty: send one object of fields per record")
	}
	return list, nil
}

// batchWrite writes the list, or refuses it whole naming the one refused.
func (s *Service) batchWrite(w http.ResponseWriter, r *http.Request, action string, writes []records.Writing) {
	typ := r.PathValue("type")
	if typ == records.BlockType {
		web.WriteError(w, errors.New("blocks are written one at a time, each checked as it is: POST /api/block with one block"))
		return
	}
	recs, notes, entry, at, err := records.WriteAllAs(s.app.Store, Agent(r).As(), action, typ, writes)
	if err != nil {
		var ve *schema.ValidationError
		if errors.As(err, &ve) {
			ve.Problems = prefixed(at, ve.Problems)
			ve.Expects = prefixed(at, ve.Expects)
			web.WriteError(w, ve)
			return
		}
		web.WriteError(w, fmt.Errorf("nothing was written: record %d could not be: %v", at, err))
		return
	}
	type made struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Page  string `json:"page"`
	}
	out := make([]made, len(recs))
	for i, rec := range recs {
		out[i] = made{rec.ID, s.apiTitle(rec), relate.Page(typ, rec.ID)}
	}
	status := http.StatusOK
	if action == "created" {
		status = http.StatusCreated
	}
	web.WriteJSON(w, status, map[string]any{"type": typ, "count": len(recs), "records": out, "notes": notes,
		"undo": undoOf(entry)}) // api_undo_said.go: one change, one Undo for all
}

// prefixed names each field by the record it is in, as [2].title.
func prefixed(at int, m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		out[fmt.Sprintf("[%d].%s", at, k)] = v
	}
	return out
}

// apiUpdateMany is PATCH /api/{type}: {"ids": [...], "fields": {...}},
// the same change to each.
func (s *Service) apiUpdateMany(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs    []string       `json:"ids"`
		Fields map[string]any `json:"fields"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&body); err != nil || len(body.IDs) == 0 || len(body.Fields) == 0 {
		web.WriteError(w, errors.New(`PATCH /api/{type} takes {"ids": [the records to change], "fields": {what to change on each}}`))
		return
	}
	writes := make([]records.Writing, len(body.IDs))
	for i, id := range body.IDs {
		own := map[string]any{}
		for k, v := range body.Fields {
			own[k] = v
		}
		writes[i] = records.Writing{ID: id, Fields: own}
	}
	s.batchWrite(w, r, "updated", writes)
}
