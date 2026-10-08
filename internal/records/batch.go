package records

// A batch is one entry for many records: an import, a sync, a meeting
// written up, writing organised or suggested. It is kept on its entry as
// each record's type, id and what it was, and taken back as one.

// Batch is how a batch is kept on its entry: for each record, its type,
// its id, and its fields before, absent for a record the batch made.
func Batch(changes []BatchItem) map[string]any {
	list := make([]any, 0, len(changes))
	for _, c := range changes {
		item := map[string]any{"type": c.Type, "id": c.ID}
		if c.Before != nil {
			item["before"] = c.Before
		}
		list = append(list, item)
	}
	return map[string]any{"changes": list}
}

// BatchItem is one record in a batch and what it was before it.
type BatchItem struct {
	Type, ID string
	Before   map[string]any
}

// BatchOf is the records a batch entry's before keeps, as Batch wrote
// them.
func BatchOf(before map[string]any) []BatchItem {
	list, _ := before["changes"].([]any)
	var out []BatchItem
	for _, v := range list {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		id, _ := m["id"].(string)
		was, _ := m["before"].(map[string]any)
		if typ != "" && id != "" {
			out = append(out, BatchItem{typ, id, was})
		}
	}
	return out
}
