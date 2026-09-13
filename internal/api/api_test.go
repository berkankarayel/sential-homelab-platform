package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulse/internal/api"
	"pulse/internal/collector"
	"pulse/internal/storage"
)

type fakeStore struct {
	all    []collector.Result
	allErr error
	byName map[string]collector.Result
}

func (f *fakeStore) LatestAll(ctx context.Context) ([]collector.Result, error) {
	return f.all, f.allErr
}

func (f *fakeStore) LatestByName(ctx context.Context, name string) (collector.Result, error) {
	r, ok := f.byName[name]
	if !ok {
		return collector.Result{}, storage.ErrNotFound
	}
	return r, nil
}

func TestHandleStatusAll(t *testing.T) {
	store := &fakeStore{
		all: []collector.Result{
			{CollectorName: "healthcheck:a", Timestamp: time.Now(), Status: collector.StatusUp},
		},
	}
	srv := api.New(store)

	rec := doRequest(t, srv, http.MethodGet, "/status")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got []map[string]any
	decodeBody(t, rec, &got)
	if len(got) != 1 || got[0]["collector_name"] != "healthcheck:a" {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestHandleStatusAll_StoreError(t *testing.T) {
	store := &fakeStore{allErr: errors.New("db down")}
	srv := api.New(store)

	rec := doRequest(t, srv, http.MethodGet, "/status")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleStatusByName_Found(t *testing.T) {
	store := &fakeStore{byName: map[string]collector.Result{
		"healthcheck:a": {
			CollectorName: "healthcheck:a",
			Status:        collector.StatusDown,
			Err:           errors.New("timeout"),
		},
	}}
	srv := api.New(store)

	rec := doRequest(t, srv, http.MethodGet, "/status/healthcheck:a")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got map[string]any
	decodeBody(t, rec, &got)
	if got["status"] != "down" {
		t.Errorf("status field = %v, want %q", got["status"], "down")
	}
	if got["error"] != "timeout" {
		t.Errorf("error field = %v, want %q", got["error"], "timeout")
	}
}

func TestHandleStatusByName_NotFound(t *testing.T) {
	srv := api.New(&fakeStore{})

	rec := doRequest(t, srv, http.MethodGet, "/status/missing")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func doRequest(t *testing.T, srv *api.Server, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
}
