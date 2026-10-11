package store_test

import "testing"

type keyA struct{}

// What is kept for one store is that store's alone, made once, and gone
// when it closes.
func TestKeptIsOneStoresAloneAndGoesOnClose(t *testing.T) {
	t.Parallel()
	a, b := open(t), open(t)
	made := 0
	mk := func() any { made++; return &made }
	first := a.Keep(keyA{}, mk)
	if again := a.Keep(keyA{}, mk); again != first || made != 1 {
		t.Errorf("kept twice under one key: made %d times", made)
	}
	if b.Kept(keyA{}) != nil {
		t.Error("another store open at the same time shares what one keeps")
	}
	if a.Kept(keyA{}) != first {
		t.Error("Kept does not find what Keep kept")
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if a.Kept(keyA{}) != nil {
		t.Error("what was kept stays after the store closes")
	}
}
