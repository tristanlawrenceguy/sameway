package app

import (
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
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

// adopt adds to this workspace what a content type from another computer
// has that this one lacks: the whole type, or the fields it is missing,
// written to schema/ and the tables the way the assistant adds them. It
// never changes or removes what is here.
func (a *App) adopt(t *schema.Type) error {
	have, ok := a.Types.Get(t.Name)
	if !ok {
		_, err := a.AddType(t)
		return err
	}
	for _, f := range t.Fields {
		if _, has := have.Field(f.Name); !has {
			if _, err := a.AddField(t.Name, f); err != nil {
				return err
			}
		}
	}
	return nil
}
