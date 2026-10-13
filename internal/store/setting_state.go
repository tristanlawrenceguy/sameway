package store

import "encoding/json"

// A few of the workspace's settings are the person's, not the computer's:
// the time zone their days are in, above all. Those travel between the
// copies the way a record does, each as a record of SettingType named
// after its key, so a zone chosen on the laptop is the desktop's too, and
// when two are chosen at once the latest wins on both. Which settings may
// travel is the app's to say (app/share.go), and a copy only ever takes
// those: a stamp naming any other is kept and never applied.

// SettingType is what a setting that travels is stamped under.
const SettingType = "_setting"

// StampSetting stamps a setting's value when it differs from the one
// stamped last. Empty is not stamped until it has been something else, so
// a copy that never set it says nothing over one that did.
func (s *Store) StampSetting(key, value string) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	var have string
	s.db.QueryRow(`SELECT value FROM _state WHERE type = ? AND id = ? AND field = 'value'`, SettingType, key).Scan(&have)
	if have == string(raw) || have == "" && value == "" {
		return
	}
	clock := s.clock.next(s.origin)
	s.db.Exec(`INSERT OR REPLACE INTO _state (type, id, field, value, clock) VALUES (?, ?, 'value', ?, ?)`, SettingType, key, string(raw), clock)
	s.db.Exec(`INSERT OR REPLACE INTO _seen (origin, clock) VALUES (?, ?)`, s.origin, clock)
}

// adoptSetting offers a setting another copy changed to OnSetting.
func (s *Store) adoptSetting(key string) {
	var raw string
	s.db.QueryRow(`SELECT value FROM _state WHERE type = ? AND id = ? AND field = 'value'`, SettingType, key).Scan(&raw)
	var value string
	if s.OnSetting == nil || json.Unmarshal([]byte(raw), &value) != nil {
		return
	}
	s.OnSetting(key, value)
}
