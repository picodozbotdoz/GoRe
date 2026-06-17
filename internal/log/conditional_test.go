package log

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/gore/internal/config"
)

func TestParseConditionEmpty(t *testing.T) {
	c, err := ParseCondition("")
	if err != nil {
		t.Fatal(err)
	}
	if c != nil {
		t.Fatal("empty condition should return nil")
	}
}

func TestParseConditionStatusGTE(t *testing.T) {
	c, err := ParseCondition("$status >= 400")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("condition should not be nil")
	}
	if c.Variable != "status" {
		t.Errorf("Variable = %q, want status", c.Variable)
	}
	if c.Operator != ">=" {
		t.Errorf("Operator = %q, want >=", c.Operator)
	}
	if c.Value != "400" {
		t.Errorf("Value = %q, want 400", c.Value)
	}
}

func TestParseConditionStatusEQ(t *testing.T) {
	c, err := ParseCondition("$status = 200")
	if err != nil {
		t.Fatal(err)
	}
	if c.Variable != "status" {
		t.Errorf("Variable = %q, want status", c.Variable)
	}
	if c.Operator != "=" {
		t.Errorf("Operator = %q, want =", c.Operator)
	}
	if c.Value != "200" {
		t.Errorf("Value = %q, want 200", c.Value)
	}
}

func TestParseConditionMethodEQ(t *testing.T) {
	c, err := ParseCondition("$request_method = POST")
	if err != nil {
		t.Fatal(err)
	}
	if c.Variable != "request_method" {
		t.Errorf("Variable = %q, want request_method", c.Variable)
	}
	if c.Operator != "=" {
		t.Errorf("Operator = %q, want =", c.Operator)
	}
	if c.Value != "POST" {
		t.Errorf("Value = %q, want POST", c.Value)
	}
}

func TestParseConditionStatusLTE(t *testing.T) {
	c, err := ParseCondition("$status <= 299")
	if err != nil {
		t.Fatal(err)
	}
	if c.Operator != "<=" {
		t.Errorf("Operator = %q, want <=", c.Operator)
	}
}

func TestParseConditionInvalid(t *testing.T) {
	_, err := ParseCondition("invalid")
	if err == nil {
		t.Error("expected error for invalid condition")
	}
}

func TestShouldLogEmptyCondition(t *testing.T) {
	entry := accessEntry{Status: 200}
	if !ShouldLog(nil, entry) {
		t.Error("nil condition should always log")
	}
}

func TestShouldLogStatusGTE400(t *testing.T) {
	c, _ := ParseCondition("$status >= 400")

	tests := []struct {
		status int
		want   bool
	}{
		{200, false},
		{399, false},
		{400, true},
		{500, true},
	}

	for _, tt := range tests {
		entry := accessEntry{Status: tt.status}
		got := ShouldLog(c, entry)
		if got != tt.want {
			t.Errorf("status %d: ShouldLog = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestShouldLogStatusEQ200(t *testing.T) {
	c, _ := ParseCondition("$status = 200")

	tests := []struct {
		status int
		want   bool
	}{
		{200, true},
		{201, false},
		{404, false},
	}

	for _, tt := range tests {
		entry := accessEntry{Status: tt.status}
		got := ShouldLog(c, entry)
		if got != tt.want {
			t.Errorf("status %d: ShouldLog = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestShouldLogMethodEQ(t *testing.T) {
	c, _ := ParseCondition("$request_method = POST")

	tests := []struct {
		method string
		want   bool
	}{
		{"POST", true},
		{"GET", false},
		{"PUT", false},
	}

	for _, tt := range tests {
		entry := accessEntry{Request: tt.method + " /test HTTP/1.1"}
		got := ShouldLog(c, entry)
		if got != tt.want {
			t.Errorf("method %s: ShouldLog = %v, want %v", tt.method, got, tt.want)
		}
	}
}

func TestAccessLogConditionalEnabled(t *testing.T) {
	cfg := `modules:
  access_log:
    enabled: true
    conditional_log: "$status >= 400"
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Modules.AccessLog.ConditionalLog != "$status >= 400" {
		t.Errorf("ConditionalLog = %q, want $status >= 400", c.Modules.AccessLog.ConditionalLog)
	}
}

func TestAccessMiddlewareConditionalLog(t *testing.T) {
	var buf bytes.Buffer

	logger := newAccessLogger(&AccessLogConfig{
		Enabled:       true,
		ConditionalLog: "$status >= 400",
	})
	if logger == nil {
		t.Fatal("logger should not be nil")
	}
	logger.writer = &buf

	entry := accessEntry{Status: 200}
	if logger.shouldLog(entry) {
		t.Error("status 200 should be filtered by $status >= 400")
	}

	buf.Reset()
	entry.Status = 404
	if !logger.shouldLog(entry) {
		t.Error("status 404 should be logged by $status >= 400")
	}
}

func TestAccessMiddlewareConditionalFiltering(t *testing.T) {
	var buf bytes.Buffer

	middleware := AccessMiddleware(true, "", "", false, "")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	handler := middleware(inner)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	_ = buf
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestAccessLogConditionalLogMethod(t *testing.T) {
	var buf bytes.Buffer

	logger := newAccessLogger(&AccessLogConfig{
		Enabled:       true,
		ConditionalLog: "$request_method = GET",
	})
	if logger == nil {
		t.Fatal("logger should not be nil")
	}
	logger.writer = &buf

	entry := accessEntry{Request: "GET /test HTTP/1.1"}
	if !logger.shouldLog(entry) {
		t.Error("GET request should be logged")
	}

	buf.Reset()
	entry.Request = "POST /test HTTP/1.1"
	if logger.shouldLog(entry) {
		t.Error("POST request should not be logged")
	}
}

func TestParseConditionStatusLT(t *testing.T) {
	c, err := ParseCondition("$status < 300")
	if err != nil {
		t.Fatal(err)
	}
	if c.Operator != "<" {
		t.Errorf("Operator = %q, want <", c.Operator)
	}
}

func TestParseConditionStatusGT(t *testing.T) {
	c, err := ParseCondition("$status > 500")
	if err != nil {
		t.Fatal(err)
	}
	if c.Operator != ">" {
		t.Errorf("Operator = %q, want >", c.Operator)
	}
}

func TestShouldLogStatusLT(t *testing.T) {
	c, _ := ParseCondition("$status < 300")

	tests := []struct {
		status int
		want   bool
	}{
		{200, true},
		{299, true},
		{300, false},
		{404, false},
	}

	for _, tt := range tests {
		entry := accessEntry{Status: tt.status}
		got := ShouldLog(c, entry)
		if got != tt.want {
			t.Errorf("status %d: ShouldLog = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestShouldLogStatusGT(t *testing.T) {
	c, _ := ParseCondition("$status > 500")

	tests := []struct {
		status int
		want   bool
	}{
		{200, false},
		{500, false},
		{501, true},
		{503, true},
	}

	for _, tt := range tests {
		entry := accessEntry{Status: tt.status}
		got := ShouldLog(c, entry)
		if got != tt.want {
			t.Errorf("status %d: ShouldLog = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestShouldLogStatusNE(t *testing.T) {
	c, _ := ParseCondition("$status != 200")

	tests := []struct {
		status int
		want   bool
	}{
		{200, false},
		{201, true},
		{404, true},
	}

	for _, tt := range tests {
		entry := accessEntry{Status: tt.status}
		got := ShouldLog(c, entry)
		if got != tt.want {
			t.Errorf("status %d: ShouldLog = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestParseConditionStatusNE(t *testing.T) {
	c, err := ParseCondition("$status != 200")
	if err != nil {
		t.Fatal(err)
	}
	if c.Operator != "!=" {
		t.Errorf("Operator = %q, want !=", c.Operator)
	}
}

func TestConditionalLogIntegration(t *testing.T) {
	cfg := `modules:
  access_log:
    enabled: true
    conditional_log: "$status >= 400"
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	middleware := AccessMiddleware(
		c.Modules.AccessLog.Enabled,
		c.Modules.AccessLog.GetOutput(),
		c.Modules.AccessLog.GetFormat(),
		c.Modules.AccessLog.Subrequest,
		c.Modules.AccessLog.GetConditionalLog(),
	)

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

func TestAccessLogConditionalLogDisabled(t *testing.T) {
	logger := newAccessLogger(&AccessLogConfig{
		Enabled:       true,
		ConditionalLog: "",
	})
	if logger == nil {
		t.Fatal("logger should not be nil")
	}

	entry := accessEntry{Status: 200}
	if !logger.shouldLog(entry) {
		t.Error("empty condition should always log")
	}
}

func TestConditionalLogInvalidCondition(t *testing.T) {
	logger := newAccessLogger(&AccessLogConfig{
		Enabled:       true,
		ConditionalLog: "invalid",
	})
	if logger == nil {
		t.Fatal("logger should not be nil")
	}

	entry := accessEntry{Status: 200}
	if !logger.shouldLog(entry) {
		t.Error("invalid condition should fall back to always log")
	}
}

func TestConditionalLogFormatString(t *testing.T) {
	var buf bytes.Buffer
	logger := &accessLogger{
		format: "$remote_addr $status",
		writer: &buf,
	}

	entry := accessEntry{
		RemoteAddr: "10.0.0.1",
		Status:     404,
	}

	logger.log(entry)

	output := buf.String()
	if !strings.Contains(output, "10.0.0.1") {
		t.Errorf("output missing remote_addr: %s", output)
	}
	if !strings.Contains(output, "404") {
		t.Errorf("output missing status: %s", output)
	}
}
