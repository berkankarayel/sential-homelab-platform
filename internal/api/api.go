// Package api exposes an HTTP REST API to query the latest collector
// results stored by storage.Store.
//
// TODO:
//   - GET /status        -> storage.LatestAll(ctx), JSON-encoded
//   - GET /status/{name} -> single collector's latest result (404 if unknown)
//   - Wire Handler() into an *http.Server in cmd/pulse/main.go
//   - Basic request logging / recover-from-panic middleware
package api

import (
	"net/http"

	"pulse/internal/storage"
)

// Server holds the dependencies the HTTP API needs.
type Server struct {
	// TODO: store *storage.Store
}

// New creates an API server backed by the given store.
func New(store *storage.Store) *Server {
	panic("TODO: implement")
}

// Handler returns the http.Handler to mount on an http.Server.
func (s *Server) Handler() http.Handler {
	panic("TODO: implement")
}
