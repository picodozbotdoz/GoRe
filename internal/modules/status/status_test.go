package status

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusEndpoint(t *testing.T) {
	handler := NewHandler("/status")

	req := httptest.NewRequest("GET", "/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Active connections:") {
		t.Error("missing Active connections")
	}
	if !strings.Contains(body, "server accepts handled requests") {
		t.Error("missing server accepts handled requests")
	}
}

func TestStatusTracksRequests(t *testing.T) {
	collector := Get()
	before := collector.Requests.Load()

	collector.ReqStart()
	collector.ReqDone()

	after := collector.Requests.Load()
	if after-before != 1 {
		t.Errorf("requests not incremented: before=%d after=%d", before, after)
	}
}

func TestStatusNotFound(t *testing.T) {
	handler := NewHandler("/status")

	req := httptest.NewRequest("GET", "/other", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
