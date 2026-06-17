# Phase 3: Remaining Directive Gaps — Implementation Plan

> **For agentic workers:** Use compose:subagent to implement task-by-task.

**Goal:** Clear ~20 remaining medium/large directive gaps. Skip: `set`, `if` (variable system — massive scope), `ssl_engine`, `ssl_conf_command`, `open_log_file_cache` (Go stdlib handles better).

**Architecture:** Same patterns as Phase 1/2. Config fields in `config.go`, module logic in `internal/modules/` or `internal/proxy/`, wire into chain.go/server.go.

---

## Batch A: SSL Medium (5 directives)

Config fields go in `TLS` struct in `internal/config/config.go`. Wire in `buildTLSConfig()` in `internal/server/server.go`.

### Task A.1: ssl_ecdh_curve
Config: `ECDHCurve string` in TLS struct.
Wire: `tlsConfig.CurvePreferences` with parsed curve IDs.

### Task A.2: ssl_dhparam
Config: `DHParam string` in TLS struct (path to DH params file).
Wire: load DH params file, set `tlsConfig.DHParams`.

### Task A.3: ssl_crl
Config: `CRL string` in TLS struct (path to CRL file).
Wire: load CRL, create `x509.RevocationList`, set `VerifyPeerCertificate`.

### Task A.4: ssl_password_file
Config: `PasswordFile string` in TLS struct.
Wire: read password file, use for key decryption via `tls.LoadX509KeyPair` with password callback.

### Task A.5: ssl_early_data
Config: `EarlyData bool` in TLS struct.
Wire: `tlsConfig.MaxEarlyData` for TLS 1.3 0-RTT.

---

## Batch B: Proxy Store (3 directives)

### Task B.1: proxy_store
Config: `Store string` in Proxy struct (file path or "off").
In ModifyResponse: write response body to file.

### Task B.2: proxy_store_access
Config: `StoreAccess string` in Proxy struct.
Set file permissions when writing stored responses.

### Task B.3: proxy_temp_file_write_size
Config: `TempFileWriteSize string` in Proxy struct.
Limit write chunk size to temp files.

---

## Batch C: Upstream (3 directives)

### Task C.1: resolve
Config: `Resolve bool` in Upstream struct.
Wire: use custom `net.Resolver` in transport dialer.

### Task C.2: slow_start
Config: `SlowStart int` in UpstreamServer struct (seconds).
Implement: gradually increase weight from 0 to configured weight over SlowStart seconds.

### Task C.3: zone
Config: `Zone string` in Upstream struct.
Document as shared memory zone name (Go uses process memory, no shared zones needed — mark as no-op with comment).

---

## Batch D: Proxy Cache Path (1 directive)

### Task D.1: proxy_cache_path
Config: `CachePath string` in CacheConfig (directory path).
Wire: persist cache entries to disk, load on startup.

---

## Batch E: Proxy Misc (2 directives)

### Task E.1: proxy_upstream
Config: `DynamicUpstream string` in Proxy struct.
Implement: dynamic upstream selection based on variable.

### Task E.2: proxy_ssl_session_ticket_key
Config: `SessionTicketKey string` in ProxySSL struct.
Wire: custom session ticket key for upstream TLS.

---

## Batch F: SSL Stapling (2 directives)

### Task F.1: ssl_stapling, ssl_stapling_verify
Config: `Stapling bool`, `StaplingVerify bool` in TLS struct.
Implement: OCSP stapling via `tls.Config.VerifyPeerCertificate` callback.

---

## Batch G: Logging (1 directive)

### Task G.1: conditional_log
Config: `ConditionalLog string` in AccessLogConfig.
Implement: log based on variable condition (e.g., `$status >= 400`).

---

## Verification

After all batches:
- `go test -race ./...`
- `go build ./cmd/gore`
- Update `docs/directive-audit.md`
