package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulse/internal/collector"
)

func TestCollect_Up(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	h := New("healthcheck:test", srv.URL, time.Second)
	result, err := h.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Status != collector.StatusUp {
		t.Errorf("Status = %v, want %v", result.Status, collector.StatusUp)
	}
	if code, _ := result.Data["status_code"].(int); code != http.StatusOK {
		t.Errorf("Data[status_code] = %v, want %d", result.Data["status_code"], http.StatusOK)
	}
}

func TestCollect_Down_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	h := New("healthcheck:test", srv.URL, time.Second)
	result, err := h.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if result.Status != collector.StatusDown {
		t.Errorf("Status = %v, want %v", result.Status, collector.StatusDown)
	}
}

func TestCollect_Down_ConnectionRefused(t *testing.T) {
	// Port 0 on loopback is never listening, so this always fails to connect.
	h := New("healthcheck:test", "http://127.0.0.1:0", time.Second)
	result, err := h.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned an error, want nil (transport errors belong in the result): %v", err)
	}
	if result.Status != collector.StatusDown {
		t.Errorf("Status = %v, want %v", result.Status, collector.StatusDown)
	}
	if result.Err == nil {
		t.Error("Err = nil, want the connection error")
	}
}

func TestNameAndInterval(t *testing.T) {
	h := New("healthcheck:test", "http://example.com", 30*time.Second)
	if got := h.Name(); got != "healthcheck:test" {
		t.Errorf("Name() = %q, want %q", got, "healthcheck:test")
	}
	if got := h.Interval(); got != 30*time.Second {
		t.Errorf("Interval() = %v, want %v", got, 30*time.Second)
	}
}
