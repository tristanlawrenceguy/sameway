package web

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// MayChange says whether whoever asked may change the workspace: not one
// who may only look, nor the internet reading what is published.
func MayChange(r *http.Request) bool {
	a := records.VisitorOf(r.Context()).Access
	return a != records.View && a != records.Public
}
