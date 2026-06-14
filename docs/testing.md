# Testing

GoRe includes unit tests and nginx compatibility tests.

## Running Tests

```bash
# All tests
go test ./...

# Verbose output
go test ./... -v

# Specific package
go test ./internal/modules/access/ -v

# With race detector
go test -race ./...
```

## Test Structure

```
gore/
├── compat_test.go              # Nginx compatibility tests
├── internal/
│   ├── config/config_test.go   # Config parsing tests
│   ├── modules/
│   │   ├── access/access_test.go
│   │   ├── gzip/gzip_test.go
│   │   ├── headers/headers_test.go
│   │   ├── ratelimit/ratelimit_test.go
│   │   └── static/static_test.go
│   ├── proxy/proxy_test.go     # Proxy tests
│   └── router/router_test.go   # Router tests
```

## Nginx Compatibility Tests

`compat_test.go` tests scenarios from nginx test suite:

| Test | nginx Test | Coverage |
|------|------------|----------|
| TestAccessModule | access.t | IP allow/deny rules |
| TestGzipModule | gzip.t | Compression, Accept-Encoding |
| TestHeadersModule | headers.t | Add/remove headers |
| TestProxyModule | proxy.t | Reverse proxy, load balancing |
| TestStaticModule | static.t | File serving, directory traversal |
| TestRewriteModule | rewrite.t | Redirects, return codes |
| TestLimitReqModule | limit_req.t | Rate limiting |

## Writing Tests

### Unit Test Pattern

```go
func TestMyFeature(t *testing.T) {
    // Setup
    handler := mymodule.New(config)
    
    // Request
    req := httptest.NewRequest("GET", "/path", nil)
    w := httptest.NewRecorder()
    
    // Execute
    handler.ServeHTTP(w, req)
    
    // Assert
    if w.Code != 200 {
        t.Errorf("status = %d, want 200", w.Code)
    }
}
```

### Integration Test Pattern

```go
func TestIntegration(t *testing.T) {
    // Start backend
    backend := httptest.NewServer(handler)
    defer backend.Close()
    
    // Start GoRe server
    cfg := &config.Config{...}
    srv := server.New(cfg)
    go srv.Start()
    defer srv.Stop(context.Background())
    
    // Test
    resp, _ := http.Get("http://localhost:8080/")
    // Assert
}
```

## Coverage

Check test coverage:

```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```
