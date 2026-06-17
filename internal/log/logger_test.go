package log

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{level: LevelInfo, output: &buf}

	l.Debugf("debug msg")
	if buf.Len() > 0 {
		t.Error("debug should not appear at info level")
	}

	l.Infof("info msg")
	if !strings.Contains(buf.String(), "info msg") {
		t.Error("info should appear at info level")
	}

	buf.Reset()
	l.Errorf("error msg")
	if !strings.Contains(buf.String(), "error msg") {
		t.Error("error should appear at info level")
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"error", LevelError},
		{"", LevelInfo},
		{"bogus", LevelInfo},
	}
	for _, tt := range tests {
		if got := parseLevel(tt.input); got != tt.want {
			t.Errorf("parseLevel(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestAccessLogOutput(t *testing.T) {
	var buf bytes.Buffer

	al := &accessLogger{format: defaultAccessFormat, writer: &buf}

	entry := accessEntry{
		RemoteAddr:    "192.168.1.1",
		RemoteUser:    "-",
		TimeLocal:     time.Now().Format("02/Jan/2006:15:04:05 -0700"),
		Request:       "GET /api/users HTTP/2.0",
		Status:        200,
		BodyBytes:     1234,
		RequestTime:   0.015,
		HTTPReferer:   "-",
		HTTPUserAgent: "curl/8.0",
	}

	al.log(entry)

	output := buf.String()
	if !strings.Contains(output, "192.168.1.1") {
		t.Errorf("output missing remote_addr: %s", output)
	}
	if !strings.Contains(output, "GET /api/users HTTP/2.0") {
		t.Errorf("output missing request: %s", output)
	}
	if !strings.Contains(output, "200") {
		t.Errorf("output missing status: %s", output)
	}
	if !strings.Contains(output, "1234") {
		t.Errorf("output missing body_bytes: %s", output)
	}
	if !strings.Contains(output, "curl/8.0") {
		t.Errorf("output missing user_agent: %s", output)
	}
}

func TestResponseWriterStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, status: 200}

	rw.WriteHeader(404)
	if rw.status != 404 {
		t.Errorf("status = %d, want 404", rw.status)
	}

	n, err := rw.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("bytes written = %d, want 5", n)
	}
	if rw.bytes != 5 {
		t.Errorf("total bytes = %d, want 5", rw.bytes)
	}
}

func TestAccessMiddlewareDisabled(t *testing.T) {
	middleware := AccessMiddleware(false, "", "", false, "")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	handler := middleware(inner)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestAccessMiddlewareCapturesRequest(t *testing.T) {
	var buf bytes.Buffer

	middleware := AccessMiddleware(true, "", "", false, "")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte("created"))
	})

	handler := middleware(inner)
	req := httptest.NewRequest("POST", "/api/items", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Referer", "http://example.com")
	req.RemoteAddr = "10.0.0.1:12345"

	rec := httptest.NewRecorder()

	_ = buf
	handler.ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Errorf("status = %d, want 201", rec.Code)
	}
}
