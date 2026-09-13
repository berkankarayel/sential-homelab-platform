// Package api exposes an HTTP REST API to query the latest collector
// results stored by storage.Store.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"pulse/internal/collector"
	"pulse/internal/storage"
)

// resultStore is the subset of *storage.Store the API depends on, so
// handlers can be tested against a fake without a real database.
type resultStore interface {
	LatestAll(ctx context.Context) ([]collector.Result, error)
	LatestByName(ctx context.Context, name string) (collector.Result, error)
}

// Server holds the dependencies the HTTP API needs.
type Server struct {
	store resultStore
}

// New creates an API server backed by the given store.
func New(store resultStore) *Server {
	return &Server{store: store}
}

// Handler returns the http.Handler to mount on an http.Server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", s.handleStatusAll)
	mux.HandleFunc("GET /status/{name}", s.handleStatusByName)
	return recoverMiddleware(logMiddleware(mux))
}

// handleStatusAll handles GET /status, returning the latest result for
// every collector that has ever reported in.
func (s *Server) handleStatusAll(w http.ResponseWriter, r *http.Request) {
	results, err := s.store.LatestAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	dtos := make([]resultDTO, len(results))
	for i, result := range results {
		dtos[i] = toDTO(result)
	}
	writeJSON(w, http.StatusOK, dtos)
}

// handleStatusByName handles GET /status/{name}, returning 404 if that
// collector has never reported a result.
func (s *Server) handleStatusByName(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	result, err := s.store.LatestByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(result))
}

// resultDTO is the JSON shape returned by the API. collector.Result isn't
// encoded directly because its Err field is an error interface, which the
// standard encoder can't turn into a useful message.
type resultDTO struct {
	CollectorName string         `json:"collector_name"`
	Timestamp     time.Time      `json:"timestamp"`
	Status        string         `json:"status"`
	Data          map[string]any `json:"data,omitempty"`
	Error         string         `json:"error,omitempty"`
}

func toDTO(r collector.Result) resultDTO {
	dto := resultDTO{
		CollectorName: r.CollectorName,
		Timestamp:     r.Timestamp,
		Status:        string(r.Status),
		Data:          r.Data,
	}
	if r.Err != nil {
		dto.Error = r.Err.Error()
	}
	return dto
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("api: panic: %v", rec)
				writeError(w, http.StatusInternalServerError, errors.New("internal error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
