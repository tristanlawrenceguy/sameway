package server

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/cloudsync"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Where keeping in step stands is said in words on Workspaces, and what
// stops it is the job's error on Help and in GET /api/jobs too: a cloud
// app gone from this computer, or the workspace's folder gone from it, is
// never silence. Whatever stops it, what is changed here is kept, and
// passes on once it is put right: changes are stamps kept in the
// database, and a missing folder is never read as things deleted.

// syncRound reads what the others wrote, then writes what this copy did.
// It is a background job every fifteen seconds (jobs.go); what went
// wrong stays in cloudsync:err for the Workspaces page as well, and the
// time it last went well in cloudsync:at.
func (s *Server) syncRound() error {
	if s.app.Store.Meta("cloudsync:id") == "" {
		return nil
	}
	folder := s.syncFolder()
	err := fmt.Errorf("%s is not on this computer just now", s.app.Store.Meta("cloudsync:cloud"))
	if folder != "" {
		err = s.passChanges(folder)
	}
	said := ""
	if err != nil {
		said = err.Error()
	}
	s.app.Store.SetMeta("cloudsync:err", said)
	if err != nil {
		return err
	}
	s.app.Store.SetMeta("cloudsync:at", s.now().Format(time.RFC3339))
	if at, _ := time.Parse(time.RFC3339, s.app.Store.Meta("cloudsync:started")); s.now().Sub(at) > 24*time.Hour {
		s.syncStart(folder)
	}
	return nil
}

func (s *Server) passChanges(folder string) error {
	files := s.app.Workspace.FilesDir()
	n, err := cloudsync.In(s.app.Store, folder, files)
	if n > 0 {
		s.Changed()
	}
	if err == nil {
		_, err = cloudsync.Out(s.app.Store, folder, files)
	}
	return err
}

// syncState is where keeping in step through cloud stands, in a sentence
// or two: what stops it and that nothing is lost, or how many other
// computers have it open and when it was last in step.
func (s *Server) syncState(cloud string) string {
	kept := " What you change here is kept, and passes on once "
	switch e := s.app.Store.Meta("cloudsync:err"); {
	case s.syncFolder() == "":
		return cloud + " is not on this computer just now." + kept + cloud + " is back."
	case e == cloudsync.ErrGone.Error():
		return "Its folder, " + filepath.Join(cloudsync.Root, s.app.Store.Meta("cloudsync:id")) + ", is no longer in " + cloud + ": deleted there, or not back yet." + kept + "the folder is back."
	case e != "":
		return "Not in step just now: " + strings.TrimSuffix(e, ".") + "." + kept + "that is put right."
	}
	said := "No other computer has opened it yet: on the other computer, open Workspaces in Sameway and choose it under On your other computers."
	if others := len(cloudsync.Others(s.app.Store, s.syncFolder())); others > 0 {
		said = "It is open on " + schema.Count(others, "other computer") + "; a change on one reaches the others as soon as " + cloud + " has copied it, usually within a minute."
	}
	if at, err := time.Parse(time.RFC3339, s.app.Store.Meta("cloudsync:at")); err == nil {
		said += " Last in step with " + cloud + " " + lower(when.Ago(at, s.now(), s.h24())) + "."
	}
	return said
}
