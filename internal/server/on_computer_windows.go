package server

import (
	"io/fs"
	"syscall"
)

// markedHidden is whether Windows marks a file or folder hidden or the
// system's own, as desktop.ini in every folder is, which Explorer does not
// show and a search here passes over.
func markedHidden(fi fs.FileInfo) bool {
	a, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	return ok && a.FileAttributes&(syscall.FILE_ATTRIBUTE_HIDDEN|syscall.FILE_ATTRIBUTE_SYSTEM) != 0
}
