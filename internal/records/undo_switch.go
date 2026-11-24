package records

import (
	"errors"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A device switched on or off, by the owner or the assistant, is undone by
// switching it back through the home's hub: what it did in the world is
// what is reversed, not a record.

// inverseSwitch switches a device back, on when it was switched off.
func (b *Book) inverseSwitch(a *store.Record, on bool) (func() (Change, error), error) {
	id, _ := a.Fields["target_id"].(string)
	if b.Switch == nil || id == "" {
		return nil, errors.New("there is no hub here to switch it back")
	}
	return func() (Change, error) {
		c, err := b.Switch(id, on)
		if err != nil {
			return Change{}, err
		}
		return *c, nil
	}, nil
}
