package server

import "github.com/tristanlawrenceguy/sameway/internal/workspace"

// UseClouds has a test's own folders stand for the cloud folders here;
// nil gives back the real ones.
func UseClouds(c []workspace.Cloud) {
	cloudFolders = func() []workspace.Cloud { return c }
	if c == nil {
		cloudFolders = workspace.CloudFolders
	}
}

// SyncRound passes changes through the cloud folder once, now.
func (s *Server) SyncRound() { s.syncRound() }

// UnpackCopy puts a copy's files in dir, as starting from it does.
func UnpackCopy(data []byte, dir string) error { return unpackCopy(data, dir) }
