package log

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var reqStartFn func()
var reqDoneFn func()

func SetRequestTracker(startFn, doneFn func()) {
	reqStartFn = startFn
	reqDoneFn = doneFn
}

const defaultAccessFormat = `$remote_addr - [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"`

type accessLogger struct {
	format string
	writer io.Writer
	mu     sync.Mutex
}

func newAccessLogger(cfg *AccessLogConfig) *accessLogger {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	format := cfg.Format
	if format == "" {
		format = defaultAccessFormat
	}

	var w io.Writer
	output := cfg.Output
	if output == "" || output == "stdout" {
		w = os.Stdout
	} else if output == "stderr" {
		w = os.Stderr
	} else {
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "access_log: failed to open %s: %v\n", output, err)
			w = os.Stdout
		} else {
			w = f
		}
	}

	return &accessLogger{format: format, writer: w}
}

func (a *accessLogger) log(entry accessEntry) {
	line := a.format

	replacer := strings.NewReplacer(
		"$remote_addr", entry.RemoteAddr,
		"$remote_user", entry.RemoteUser,
		"$time_local", entry.TimeLocal,
		"$request", entry.Request,
		"$status", fmt.Sprintf("%d", entry.Status),
		"$body_bytes_sent", fmt.Sprintf("%d", entry.BodyBytes),
		"$request_time", fmt.Sprintf("%.3f", entry.RequestTime),
		"$http_referer", entry.HTTPReferer,
		"$http_user_agent", entry.HTTPUserAgent,
		"$http_x_forwarded_for", entry.HTTPXForwardedFor,
		"$upstream_addr", entry.UpstreamAddr,
	)

	line = replacer.Replace(line)

	a.mu.Lock()
	defer a.mu.Unlock()
	fmt.Fprintln(a.writer, line)
}

type accessEntry struct {
	RemoteAddr        string
	RemoteUser        string
	TimeLocal         string
	Request           string
	Status            int
	BodyBytes         int
	RequestTime       float64
	HTTPReferer       string
	HTTPUserAgent     string
	HTTPXForwardedFor string
	UpstreamAddr      string
}

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func AccessMiddleware(enabled bool, output, format string) func(http.Handler) http.Handler {
	if !enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	logger := newAccessLogger(&AccessLogConfig{
		Enabled: enabled,
		Output:  output,
		Format:  format,
	})
	if logger == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			if reqStartFn != nil {
				reqStartFn()
			}

			rw := &responseWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(rw, r)

			if reqDoneFn != nil {
				reqDoneFn()
			}

			duration := time.Since(start).Seconds()

			remoteAddr := r.RemoteAddr
			if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
				remoteAddr = remoteAddr[:idx]
			}

			reqStr := fmt.Sprintf("%s %s %s", r.Method, r.URL.Path, r.Proto)

			entry := accessEntry{
				RemoteAddr:        remoteAddr,
				RemoteUser:        "-",
				TimeLocal:         start.Format("02/Jan/2006:15:04:05 -0700"),
				Request:           reqStr,
				Status:            rw.status,
				BodyBytes:         rw.bytes,
				RequestTime:       duration,
				HTTPReferer:       r.Header.Get("Referer"),
				HTTPUserAgent:     r.Header.Get("User-Agent"),
				HTTPXForwardedFor: r.Header.Get("X-Forwarded-For"),
			}

			if entry.HTTPReferer == "" {
				entry.HTTPReferer = "-"
			}
			if entry.HTTPUserAgent == "" {
				entry.HTTPUserAgent = "-"
			}

			logger.log(entry)
		})
	}
}
