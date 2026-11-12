package server

import "testing"

// The knock is real: a phone there refuses a knock at a closed port,
// which says it is there, on Windows as anywhere.
func TestAPhoneThereIsKnownByItsRefusal(t *testing.T) {
	t.Parallel()
	if !knockAt("127.0.0.1") {
		t.Errorf("this computer, knocked at, refuses: there")
	}
}
