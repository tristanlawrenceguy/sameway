package app

import (
	"slices"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// share says what travels when other computers host this workspace too.
// Each person's chats and the assistant's questions stay, and so do the
// programs a button runs, the devices on this network and the files kept
// in this folder; everything else is kept the same everywhere, content
// types included, which only ever grow.
func (a *App) share() {
	a.Store.Local = map[string]bool{"message": true, "conversation": true, "proposal": true, "action": true, "device": true, "file": true}
	a.Store.LocalRecord = chat.LocalEntry(a.Store.Local)
	a.Store.OnSchema = a.adopt
}

// adopt makes a content type here what the other computers that host the
// workspace have: made, deleted, hidden or shown, its fields added, given
// choices, labelled, hidden or deleted, through the same changes a person
// asks for here. Choices only ever join; for the rest, the latest wins.
func (a *App) adopt(sc *store.Schema) error {
	t := sc.Type
	have, ok := a.Types.Get(t.Name)
	switch {
	case sc.Gone:
		if ok {
			return a.RemoveType(t.Name)
		}
		return nil
	case !ok:
		_, err := a.AddType(t)
		return err
	}
	if have.Hidden != t.Hidden {
		if _, err := a.SetHidden(t.Name, "", t.Hidden); err != nil {
			return err
		}
	}
	for _, name := range sc.Deleted {
		if _, has := have.Field(name); has {
			if _, err := a.RemoveField(t.Name, name); err != nil {
				return err
			}
		}
	}
	for _, f := range t.Fields {
		if err := a.adoptField(have, f); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) adoptField(have *schema.Type, f schema.Field) error {
	mine, ok := have.Field(f.Name)
	if !ok {
		_, err := a.AddField(have.Name, f)
		return err
	}
	for _, v := range f.Values {
		if !slices.Contains(mine.Values, v) {
			if _, err := a.AddChoice(have.Name, f.Name, v, f.Labels[v]); err != nil {
				return err
			}
			mine, _ = have.Field(f.Name)
		} else if mine.Labels[v] != f.Labels[v] {
			if _, err := a.Relabel(have.Name, f.Name, v, f.Labels[v]); err != nil {
				return err
			}
			mine, _ = have.Field(f.Name)
		}
	}
	if mine.Label != f.Label {
		if _, err := a.Relabel(have.Name, f.Name, "", f.Label); err != nil {
			return err
		}
		mine, _ = have.Field(f.Name)
	}
	if mine.Hidden != f.Hidden {
		_, err := a.SetHidden(have.Name, f.Name, f.Hidden)
		return err
	}
	return nil
}
