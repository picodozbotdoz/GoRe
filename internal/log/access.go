package log

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Condition struct {
	Variable string
	Operator string
	Value    string
}

func ParseCondition(s string) (*Condition, error) {
	if s == "" {
		return nil, nil
	}

	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "$") {
		return nil, fmt.Errorf("condition must start with $: %s", s)
	}

	operators := []string{"!=", ">=", "<=", ">", "<", "="}
	for _, op := range operators {
		if idx := strings.Index(s, " "+op+" "); idx > 0 {
			variable := s[:idx]
			variable = strings.TrimPrefix(variable, "$")
			rest := s[idx+len(op)+2:]
			return &Condition{
				Variable: variable,
				Operator: op,
				Value:    rest,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid condition format: %s", s)
}

func ShouldLog(c *Condition, entry accessEntry) bool {
	if c == nil {
		return true
	}

	var actual string
	switch c.Variable {
	case "status":
		actual = strconv.Itoa(entry.Status)
	case "request_method":
		parts := strings.SplitN(entry.Request, " ", 2)
		if len(parts) > 0 {
			actual = parts[0]
		}
	default:
		return true
	}

	expectedInt, errExpected := strconv.Atoi(c.Value)
	actualInt, errActual := strconv.Atoi(actual)

	if errExpected == nil && errActual == nil {
		switch c.Operator {
		case "=":
			return actualInt == expectedInt
		case "!=":
			return actualInt != expectedInt
		case ">":
			return actualInt > expectedInt
		case ">=":
			return actualInt >= expectedInt
		case "<":
			return actualInt < expectedInt
		case "<=":
			return actualInt <= expectedInt
		}
	}

	switch c.Operator {
	case "=":
		return actual == c.Value
	case "!=":
		return actual != c.Value
	default:
		return true
	}
}

var (
	reqStartFn func()
	reqDoneFn  func()
)

func SetRequestTracker(startFn, doneFn func()) {
	reqStartFn = startFn
	reqDoneFn = doneFn
}

const defaultAccessFormat = `$remote_addr - [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"`

type accessLogger struct {
	format    string
	writer    io.Writer
	mu        sync.Mutex
	condition *Condition
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

	var condition *Condition
	if cfg.ConditionalLog != "" {
		var err error
		condition, err = ParseCondition(cfg.ConditionalLog)
		if err != nil {
			fmt.Fprintf(os.Stderr, "access_log: invalid condition %q: %v\n", cfg.ConditionalLog, err)
			condition = nil
		}
	}

	return &accessLogger{format: format, writer: w, condition: condition}
}

func (a *accessLogger) shouldLog(entry accessEntry) bool {
	return ShouldLog(a.condition, entry)
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
		"$http_x_subrequest_uri", entry.HTTPXSubrequestUri,
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
	HTTPXSubrequestUri string
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

func AccessMiddleware(enabled bool, output, format string, subrequest bool, conditionalLog string) func(http.Handler) http.Handler {
	if !enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	logger := newAccessLogger(&AccessLogConfig{
		Enabled:       enabled,
		Output:        output,
		Format:        format,
		Subrequest:    subrequest,
		ConditionalLog: conditionalLog,
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

			isSubrequest := r.Header.Get("X-Subrequest") != "" || r.Header.Get("X-Sub-Request") != ""

			rw := &responseWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(rw, r)

			if reqDoneFn != nil {
				reqDoneFn()
			}

			if isSubrequest && !subrequest {
				return
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
				HTTPXSubrequestUri: r.Header.Get("X-Subrequest-Uri"),
			}

			if entry.HTTPReferer == "" {
				entry.HTTPReferer = "-"
			}
			if entry.HTTPUserAgent == "" {
				entry.HTTPUserAgent = "-"
			}

			if logger.shouldLog(entry) {
				logger.log(entry)
			}
		})
	}
}
