package access

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccessAllow(t *testing.T) {
	_, network, _ := net.ParseCIDR("192.168.0.0/16")
	handler := New([]Rule{{Allow: network}})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAccessDeny(t *testing.T) {
	_, network, _ := net.ParseCIDR("0.0.0.0/0")
	handler := New([]Rule{{Deny: network}})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"192.168.0.0/16", false},
		{"10.0.0.1", false},
		{"all", false},
		{"invalid", true},
	}
	for _, tt := range tests {
		_, err := ParseCIDR(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseCIDR(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}
