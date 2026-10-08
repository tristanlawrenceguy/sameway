package server

// ReadLocal lets a test read the pages it serves on this computer, which a
// shared link never may.
func ReadLocal(on bool) { readLocal = on }
