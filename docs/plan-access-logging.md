# Implementation Plan: Access Logging

## Problem

GoRe has no request logging. Every production server needs access logs for debugging, monitoring, and analytics. This is also the foundation — many future features (error logging, metrics, conditional logging) depend on a reusable logging framework.

## Design Principles

1. **Reusable** — single `internal/log` package, imported by all modules
2. **Singleton** — package-level `log.Info()`, `log.Error()` work everywhere
3. **Middleware** — access logging wraps handlers like gzip/headers
4. **nginx-compatible** — familiar log format with `$variable` syntax
5. **AI-agent friendly** — clear package API, self-documenting, one import

## Architecture

```
internal/log/
├── logger.go      # Core logger: levels, output, format, singleton
├── access.go      # Access log middleware (HTTP handler wrapper)
└── logger_test.go # Tests
```

### Package API (logger.go)

```go
package log

// Levels
func Debug(msg string, args ...any)
func Info(msg string, args ...any)
func Warn(msg string, args ...any)
func Error(msg string, args ...any)

// Access log
func Access(req *http.Request, status int, bytes int, duration time.Duration, upstream string)

// Configuration (called once at startup)
func Init(cfg *Config)

type Config struct {
    Level      string // "debug", "info", "warn", "error"
    Output     string // "stdout", "stderr", or file path
    Format     string // "text" or "json"
    AccessLog  *AccessLogConfig
}

type AccessLogConfig struct {
    Enabled  bool
    Output   string // "stdout", "stderr", or file path
    Format   string // nginx-style format string
}
```

### Access Log Format

Default (nginx combined):
```
$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
```

Example output:
```
127.0.0.1 - - [15/Jun/2026:12:34:56 +0700] "GET /api/users HTTP/2.0" 200 1234 "-" "Mozilla/5.0"
```

### Supported Variables

| Variable | Value | Source |
|----------|-------|--------|
| `$remote_addr` | Client IP | `req.RemoteAddr` (parsed) |
| `$remote_user` | Auth user | `-` (future: auth module) |
| `$time_local` | Local time | `time.Now().Format(...)` |
| `$request` | Method + Path + Proto | `req.Method + req.URL.Path + req.Proto` |
| `$status` | Response code | Captured via `responseWriter` |
| `$body_bytes_sent` | Response size | Captured via `responseWriter` |
| `$request_time` | Request duration | `time.Since(start)` |
| `$http_referer` | Referer header | `req.Header.Get("Referer")` |
| `$http_user_agent` | User-Agent header | `req.Header.Get("User-Agent")` |
| `$http_x_forwarded_for` | X-Forwarded-For | `req.Header.Get("X-Forwarded-For")` |
| `$upstream_addr` | Upstream address | Passed by proxy module |

### Config (gore.yaml)

```yaml
modules:
  access_log:
    enabled: true
    output: stdout          # stdout, stderr, or /var/log/gore/access.log
    format: '$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"'
  error_log:
    level: info             # debug, info, warn, error
    output: stderr
```

### Middleware (access.go)

```go
// NewMiddleware wraps a handler and logs every request
func NewMiddleware(cfg *AccessLogConfig) func(http.Handler) http.Handler

// responseWriter captures status code and bytes written
type responseWriter struct {
    http.ResponseWriter
    status  int
    bytes   int
    written bool
}
```

### Integration Points

1. **modules/chain.go** — add access log middleware (outermost, wraps everything)
2. **config/config.go** — add `AccessLogConfig` and `ErrorLogConfig` to `ModulesConfig`
3. **cmd/gore/main.go** — call `log.Init(cfg)` before server starts
4. **server.go** — replace `log.Printf` with `log.Error`

## Implementation Steps

### Step 1: Core Logger (`internal/log/logger.go`)

- Global singleton: `var defaultLogger *Logger`
- `Init(cfg)` sets up output writer, level filter, format
- Level methods: `Debug`, `Info`, `Warn`, `Error`
- `Access()` method for access log entries
- Thread-safe (uses `sync.Mutex` for file writes)
- Buffer/flush for file output (line-buffered)

### Step 2: Access Log Middleware (`internal/log/access.go`)

- `NewMiddleware(cfg) func(http.Handler) http.Handler`
- `responseWriter` wrapper captures status + bytes
- Records `time.Now()` at request start
- On handler return: formats and writes access log line
- Variable substitution from request/response data

### Step 3: Config Types (`internal/config/config.go`)

Add to `ModulesConfig`:

```go
type AccessLogConfig struct {
    Enabled bool   `yaml:"enabled"`
    Output  string `yaml:"output,omitempty"`
    Format  string `yaml:"format,omitempty"`
}

type ErrorLogConfig struct {
    Level  string `yaml:"level,omitempty"`
    Output string `yaml:"output,omitempty"`
}
```

### Step 4: Wire Into Chain (`internal/modules/chain.go`)

Access log middleware is **outermost** (first to see request, last to see response):

```
Request → AccessLog → Access → RateLimit → Headers → Gzip → Handler → Response
```

### Step 5: Init in main.go

```go
log.Init(&log.Config{
    Level:  cfg.Modules.ErrorLog.GetLevel(),
    Output: cfg.Modules.ErrorLog.GetOutput(),
    AccessLog: &log.AccessLogConfig{
        Enabled: cfg.Modules.AccessLog.Enabled,
        Output:  cfg.Modules.AccessLog.GetOutput(),
        Format:  cfg.Modules.AccessLog.GetFormat(),
    },
})
```

### Step 6: Tests (`internal/log/logger_test.go`)

- Test level filtering (debug message not shown at info level)
- Test access log format output
- Test variable substitution
- Test responseWriter captures status and bytes
- Test file output with flush
- Test concurrent safety

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/log/logger.go` | **CREATE** | Core logger with levels, output, format |
| `internal/log/access.go` | **CREATE** | Access log middleware + responseWriter |
| `internal/log/logger_test.go` | **CREATE** | Unit tests |
| `internal/config/config.go` | MODIFY | Add AccessLogConfig, ErrorLogConfig |
| `internal/config/config_test.go` | MODIFY | Add config parsing tests |
| `internal/modules/chain.go` | MODIFY | Wire access log middleware |
| `cmd/gore/main.go` | MODIFY | Call log.Init() at startup |
| `internal/server/server.go` | MODIFY | Replace log.Printf with log.Error |
| `docs/configuration.md` | MODIFY | Document new config fields |

## Verification

1. `go build ./...` — compiles
2. `go vet ./...` — no issues
3. `go test ./... -race` — all tests pass, no races
4. Manual: start GoRe with access_log enabled, make requests, verify log output
5. `gitleaks detect` — no secrets leaked
