package server

// SyncRound passes changes through the cloud folder once, now, and says
// what stopped it.
func (s *Server) SyncRound() error { return s.syncRound() }

// UnpackCopy puts a copy's files in dir, as starting from it does.
func UnpackCopy(data []byte, dir string) error { return unpackCopy(data, dir) }
