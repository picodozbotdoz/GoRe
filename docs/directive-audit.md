# GoRe vs nginx Directive-Level Compatibility Audit

Cross-reference of every production-relevant nginx directive against GoRe's implementation.

## Status Legend

| Status | Meaning |
|--------|---------|
| ✅ | Fully implemented |
| 🔧 | Partially implemented (core behavior works, missing edge cases) |
| ⬜ | Config field parsed but not functional |
| ❌ | Not implemented |
| ⛔ | Out of scope for GoRe (stream/mail/geo/specialized) |

---

## ngx_http_core_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `server` | — | ✅ | ✅ | — | — |
| `listen` | `*:80` | `addr: ":80"` | ✅ | — | — |
| `listen ssl` | — | `tls:` block | ✅ | — | — |
| `server_name` | `""` | `name:` | ⬜ | Parsed but not used for virtual host routing | Medium |
| `root` | `html` | `root:` | ✅ | — | — |
| `location` | — | `path:` | 🔧 | Prefix match only; no `=`, `~`, `~*`, `^~` modifiers; no named locations (`@name`); no nested locations | Large |
| `try_files` | — | `try_files:` | 🔧 | `$uri`, `$uri/`, `=CODE`, last entry redirect. Missing: `$uri/index.html` auto, `@named` locations, complex variable substitution | Medium |
| `return` | — | `return:` | 🔧 | `CODE body`, `CODE URL`, plain URL redirect. Missing: `return 204`, `return 418`, `return` without code | Small |
| `error_page` | — | `error_page.pages` | ✅ | Custom error pages per status code | — |
| `client_max_body_size` | `1m` | `client_max_body_size` | ✅ | — | — |
| `client_body_timeout` | `60s` | `client_body_timeout` | ✅ | — | — |
| `client_header_timeout` | `60s` | `client_header_timeout` | ✅ | — | — |
| `send_timeout` | `60s` | `send_timeout` | ✅ | — | — |
| `keepalive_timeout` | `75s` | `keepalive_timeout` | ✅ | Configurable per-listen block | — |
| `keepalive_requests` | `1000` | `keepalive_requests` | ✅ | — | — |
| `sendfile` | `off` | N/A | ⛔ | Go handles this internally via `http.ServeContent` | — |
| `tcp_nopush` | `off` | N/A | ⛔ | OS-level, not applicable in Go | — |
| `tcp_nodelay` | `on` | N/A | ⛔ | OS-level, Go enables by default | — |
| `type` | `text/plain` | `default_type` | ✅ | — | — |
| `default_type` | `text/plain` | `default_type` | ✅ | — | — |
| `autoindex` | `off` | `autoindex: true/false` | ✅ | — | — |
| `alias` | — | ❌ | ❌ | Path aliasing (different from root) | Medium |
| `limit_except` | — | ❌ | ❌ | Restrict by HTTP method | Small |
| `if` | — | ❌ | ❌ | Conditional blocks (nginx discourages this) | Large |
| `internal` | — | ❌ | ❌ | Mark location as internal-only | Small |
| `satisfy` | `all` | ❌ | ❌ | Combine auth and access rules | Medium |
| `resolver` | system | ❌ | ❌ | DNS resolver for upstreams | Medium |
| `merge_slashes` | `on` | `merge_slashes` | ✅ | — | — |
| `ignore_invalid_headers` | `on` | N/A | ⛔ | Go's net/http handles this | — |
| `underscores_in_headers` | `off` | N/A | ⛔ | Go's net/http handles this | — |
| `server_tokens` | `on` | `server_tokens` | ✅ | — | — |
| `etag` | `on` | ✅ | ✅ | Weak ETags for static files | — |
| `if_modified_since` | `exact` | N/A | ⛔ | `http.ServeContent` handles this | — |
| `rewrite` | — | `rewrite:` | ✅ | Regex with $1 backreference, 301/302 codes | — |
| `set` | — | ❌ | ❌ | Variable assignment | Large |
| `map` | — | `map:` | ✅ | Regex header→header mapping | — |
| `split_clients` | — | `split_clients:` | ✅ | Percentage-based A/B testing | — |

---

## ngx_http_proxy_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `proxy_pass` | — | `proxy.upstream` | ✅ | — | — |
| `proxy_set_header` | — | `set_headers` (per-upstream) | 🔧 | Not per-location; can't use variables like `$host`, `$remote_addr` | Medium |
| `proxy_pass_request_headers` | `on` | `pass_request_headers` | ✅ | — | — |
| `proxy_pass_request_body` | `on` | `pass_request_body` | ✅ | — | — |
| `proxy_connect_timeout` | `60s` | `connect_timeout` | ✅ | — | — |
| `proxy_send_timeout` | `60s` | `send_timeout` | ✅ | — | — |
| `proxy_read_timeout` | `60s` | `read_timeout` | ✅ | — | — |
| `proxy_buffering` | `on` | `buffering: true/false` | ✅ | — | — |
| `proxy_buffer_size` | `4k/8k` | `buffer_size` | ✅ | — | — |
| `proxy_buffers` | `8 4k/8k` | ❌ | ❌ | Number and size of buffers for response body | Small |
| `proxy_busy_buffers_size` | `8k/16k` | ❌ | ❌ | Limit on response data buffered before sending | Small |
| `proxy_cache` | `off` | `cache.enabled` | ✅ | — | — |
| `proxy_cache_path` | — | ❌ | ❌ | Disk-based cache with levels/keys_zone | Large |
| `proxy_cache_valid` | — | ❌ | ❌ | Cache validity per status code | Medium |
| `proxy_cache_key` | `$scheme$proxy_host$request_uri` | ❌ | ❌ | Custom cache key | Small |
| `proxy_no_cache` | — | ❌ | ❌ | Conditions to bypass cache | Small |
| `proxy_cache_bypass` | — | ❌ | ❌ | Conditions to bypass cache | Small |
| `proxy_cache_use_stale` | — | ❌ | ❌ | Serve stale on error/timeout | Medium |
| `proxy_cache_lock` | `off` | ❌ | ❌ | Prevent cache stampede | Medium |
| `proxy_next_upstream` | `error timeout` | `next_upstream` | ✅ | Supports `error`, `timeout`, `invalid_header` flags | — |
| `proxy_next_upstream_tries` | `0` | `next_upstream_tries` | ✅ | — | — |
| `proxy_next_upstream_timeout` | `0` | `next_upstream_timeout` | ✅ | — | — |
| `proxy_upstream` | — | ❌ | ❌ | Dynamic upstream selection | Large |
| `proxy_http_version` | `1.0` | N/A | ⛔ | Go uses HTTP/1.1 by default; HTTP/2 via `ForceAttemptHTTP2` | — |
| `proxy_socket_keepalive` | `off` | ❌ | ❌ | Enable TCP keepalive on upstream connections | Small |
| `proxy_set_x_forwarded_for` | `on` | ✅ | ✅ | Via `set_headers` config | — |
| `proxy_redirect` | `default` | `redirect` | ✅ | — | — |
| `proxy_intercept_errors` | `off` | ❌ | ❌ | Intercept upstream error pages | Medium |
| `proxy_store` | `off` | ❌ | ❌ | Store upstream responses to files | Large |
| `proxy_store_access` | `user:rw ...` | ❌ | ❌ | File permissions for stored responses | Large |
| `proxy_max_temp_file_size` | `1024m` | ❌ | ❌ | Max temp file size for buffering | Medium |
| `proxy_request_buffering` | `on` | ❌ | ❌ | Buffer client request before sending upstream | Medium |
| `proxy_temp_file_write_size` | `256k/512k` | ❌ | ❌ | Size of data written to temp files | Medium |
| `proxy_ssl_verify` | `off` | ❌ | ❌ | Verify upstream TLS certificate | Small |
| `proxy_ssl_certificate` | — | ❌ | ❌ | Client certificate for upstream mTLS | Medium |
| `proxy_ssl_trusted_certificate` | — | ❌ | ❌ | Trusted CA for upstream verification | Medium |
| `proxy_ssl_protocols` | `TLSv1 TLSv1.1 TLSv1.2 TLSv1.3` | ❌ | ❌ | Upstream TLS protocol versions | Small |
| `proxy_ssl_ciphers` | `DEFAULT` | ❌ | ❌ | Upstream TLS cipher suites | Small |
| `proxy_ssl_server_name` | `off` | ❌ | ❌ | SNI for upstream connections | Small |
| `proxy_ssl_session_reuse` | `on` | ❌ | ❌ | TLS session reuse for upstream | Medium |
| `proxy_ssl_name` | `$proxy_host` | ❌ | ❌ | SNI name for upstream | Small |
| `proxy_ssl_session_ticket_key` | — | ❌ | ❌ | Custom session tickets | Large |
| `proxy_cookie_domain` | — | ❌ | ❌ | Rewrite cookie domain | Medium |
| `proxy_cookie_path` | — | ❌ | ❌ | Rewrite cookie path | Medium |
| `proxy_hide_header` | — | ❌ | ❌ | Remove response headers from upstream | Small |
| `proxy_set_header` (host) | `$proxy_host` | `set_headers` | 🔧 | Can't set Host to `$host` variable | Medium |
| `proxy_protocol` | `off` | ❌ | ❌ | PROXY protocol for upstream | Medium |
| `proxy_method` | `$request_method` | ❌ | ❌ | Override upstream request method | Small |
| `proxy_http10` | `off` | N/A | ⛔ | Go uses HTTP/1.1 by default | — |
| `proxy_buffering` (per-location) | `on` | Per-upstream | 🔧 | Not per-location | Medium |

---

## ngx_http_ssl_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `ssl_certificate` | — | `tls.cert` | ✅ | — | — |
| `ssl_certificate_key` | — | `tls.key` | ✅ | — | — |
| `ssl_protocols` | `TLSv1 TLSv1.1 TLSv1.2 TLSv1.3` | `tls.min_version` | 🔧 | Only min_version; can't disable specific protocols | Small |
| `ssl_ciphers` | `DEFAULT` | `tls.ciphers` | 🔧 | Works but HTTP/2 requires specific ciphers; some configs fail | Medium |
| `ssl_prefer_server_ciphers` | `off` | N/A | ⛔ | Go crypto/tls handles cipher ordering automatically | — |
| `ssl_session_cache` | `none` | N/A | ⛔ | Go stdlib manages session caching internally | — |
| `ssl_session_tickets` | `on` | N/A | ⛔ | Go handles ticket-based sessions automatically | — |
| `ssl_session_timeout` | `5m` | ❌ | ❌ | Session ticket lifetime | Small |
| `ssl_stapling` | `off` | ❌ | ❌ | OCSP stapling | Large |
| `ssl_stapling_verify` | `off` | ❌ | ❌ | Verify OCSP response | Large |
| `ssl_early_data` | `off` | ❌ | ❌ | 0-RTT for TLS 1.3 | Large |
| `ssl_crl` | — | ❌ | ❌ | Certificate revocation list | Medium |
| `ssl_client_certificate` | — | ❌ | ❌ | Client certificate verification | Medium |
| `ssl_verify_client` | `off` | ❌ | ❌ | Client certificate auth | Medium |
| `ssl_verify_depth` | `1` | ❌ | ❌ | Certificate chain depth | Small |
| `ssl_dhparam` | — | ❌ | ❌ | DH parameters for DHE ciphers | Medium |
| `ssl_ecdh_curve` | `auto` | ❌ | ❌ | ECDH curves | Small |
| `ssl_conf_command` | — | ❌ | ❌ | OpenSSL configuration commands | Large |
| `ssl_password_file` | — | ❌ | ❌ | Encrypted private key password file | Medium |
| `ssl_reject_handshake` | `off` | ❌ | ❌ | Reject TLS handshake | Small |
| `ssl_conf_command` | — | ❌ | ❌ | Direct OpenSSL commands | Large |
| `ssl_engine` | — | ❌ | ❌ | OpenSSL engine | Large |

---

## ngx_http_gzip_module / compression

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `gzip` | `off` | `gzip.enabled` | ✅ | — | — |
| `gzip_comp_level` | `1` | `gzip.level` | ✅ | — | — |
| `gzip_types` | `text/html` | `gzip.types` | ✅ | — | — |
| `gzip_min_length` | `20` | `gzip.min_length` | ✅ | — | — |
| `gzip_vary` | `off` | `gzip.vary` | ✅ | — | — |
| `gzip_proxied` | `off` | `gzip.proxied` | ✅ | — | — |
| `gzip_disable` | `msie6` | `gzip.disable` | ✅ | — | — |
| `gzip_static` | `off` | ❌ | ❌ | Serve pre-compressed .gz files | Medium |
| `brotli` (3rd party) | `off` | `brotli.enabled` | ✅ | — | — |
| `brotli_comp_level` | `4` | `brotli.level` | ✅ | — | — |
| `brotli_types` | `text/html text/plain text/css ...` | `brotli.types` | ✅ | — | — |
| `gunzip` | `off` | `gunzip: true` | ✅ | — | — |

---

## ngx_http_access_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `allow` | — | `access.rules.allow` | ✅ | — | — |
| `deny` | — | `access.rules.deny` | ✅ | — | — |

---

## ngx_http_limit_req_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `limit_req_zone` | — | `rate_limit.zone` | ⬜ | Zone name parsed but not used | Small |
| `limit_req` | — | `rate_limit.rate/burst` | ✅ | — | — |
| `limit_req_status` | `503` | `rate_limit.status` | ✅ | Configurable status code (default 429) | — |
| `limit_req_log_level` | `error` | `rate_limit.log_level` | ✅ | — | — |

---

## ngx_http_limit_conn_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `limit_conn_zone` | — | `limit_conn.zone` | ⬜ | Zone name parsed but not used | Small |
| `limit_conn` | — | `limit_conn.connections` | ✅ | — | — |
| `limit_conn_status` | `503` | Hardcoded `503` | ✅ | — | — |
| `limit_conn_log_level` | `error` | `limit_conn.log_level` | ✅ | — | — |

---

## ngx_http_log_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `access_log` | `logs/access.log combined` | `access_log.enabled/output/format` | ✅ | — | — |
| `log_format` | `combined` | `access_log.format` | 🔧 | Limited variable support; can't define custom formats | Medium |
| `access_log off` | — | `access_log.enabled: false` | ✅ | — | — |
| `conditional_log` | — | ❌ | ❌ | Log based on variables | Medium |
| `log_subrequest` | `on` | ❌ | ❌ | Log subrequest URIs | Small |
| `open_log_file_cache` | `off` | ❌ | ❌ | Cache log file descriptors | Large |
| `error_log` | `error` | `error_log.level` | ✅ | — | — |

---

## ngx_http_headers_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `add_header` | — | `headers.add` | ✅ | — | — |
| `expires` | `off` | `headers.expires` | ✅ | — | — |
| `add_header` (with `always`) | — | `headers.add[].always` | ✅ | — | — |

---

## ngx_http_rewrite_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `return` | — | `return:` | ✅ | — | — |
| `rewrite` | — | `rewrite:` | ✅ | — | — |
| `break` | — | ❌ | ❌ | Stop processing rewrite rules | Small |
| `if` | — | ❌ | ❌ | Conditional blocks (discouraged in nginx) | Large |
| `set` | — | ❌ | ❌ | Variable assignment | Large |
| `rewrite_log` | `off` | ❌ | ❌ | Log rewrite processing | Small |

---

## ngx_http_auth_basic_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `auth_basic` | — | `basic_auth.realm` | ✅ | — | — |
| `auth_basic_user_file` | — | `basic_auth.users` (map) | ✅ | In-memory map; no file-based user storage | Small |
| `auth_basic_user_file` (encrypted) | — | ❌ | ❌ | bcrypt/sha encrypted passwords | Medium |

---

## ngx_http_auth_request_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `auth_request` | — | `auth_request:` | ✅ | — | — |
| `auth_request_set` | — | ❌ | ❌ | Set variables from auth response | Medium |

---

## ngx_http_sub_filter_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `sub_filter` | — | `sub_filter:` | ✅ | — | — |
| `sub_filter_once` | `on` | `sub_filter_once` | ✅ | — | — |
| `sub_filter_types` | `text/html` | `sub_filter_types` | ✅ | — | — |

---

## ngx_http_mirror_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `mirror` | `off` | `mirror:` | ✅ | — | — |
| `mirror_request_body` | `on` | `mirror:` (body forwarded) | ✅ | — | — |

---

## ngx_http_realip_module

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `real_ip_header` | `X-Forwarded-For` | `real_ip.from` | ✅ | — | — |
| `set_real_ip_from` | — | `real_ip.from` (multi-CIDR) | ✅ | — | — |
| `real_ip_recursive` | `off` | `real_ip.recursive` | ✅ | — | — |

---

## ngx_http_upstream_module (load balancing)

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `upstream` | — | `upstreams:` | ✅ | — | — |
| `server` (in upstream) | — | `upstreams.servers` | ✅ | — | — |
| `least_conn` | — | `strategy: least-conn` | ✅ | — | — |
| `ip_hash` | — | `strategy: ip_hash` | ✅ | — | — |
| `hash` | — | `strategy: hash` | ✅ | Consistent hash with weighted replicas | — |
| `keepalive` | — | `keepalive` | ✅ | — | — |
| `keepalive_timeout` | `60s` | `keepalive_timeout` | ✅ | — | — |
| `keepalive_requests` | `1000` | `keepalive_requests` | ✅ | — | — |
| `health_check` | — | `health_check` | ✅ | — | — |
| `slow_start` | `0s` | ❌ | ❌ | Gradual weight increase after recovery | Large |
| `backup` | — | `servers[].backup` | ✅ | — | — |
| `down` | — | `servers[].down` | ✅ | — | — |
| `resolve` | — | ❌ | ❌ | DNS resolution for upstream | Large |
| `zone` | — | ❌ | ❌ | Shared memory zone for upstream | Large |

---

## ngx_http_ssl_module (session/reuse)

| Directive | nginx Default | GoRe | Status | Gap | Effort |
|-----------|--------------|------|--------|-----|--------|
| `ssl_session_cache` | `none` | N/A | ⛔ | Go stdlib manages session caching internally | — |
| `ssl_session_tickets` | `on` | N/A | ⛔ | Go handles ticket-based sessions automatically | — |

---

## Out of Scope (⛔)

These nginx features are explicitly out of scope for GoRe:

| Module | Directives | Reason |
|--------|-----------|--------|
| `ngx_stream_*` | All stream directives | L4 proxying not in GoRe's scope |
| `ngx_mail_*` | All mail directives | Mail proxying not in GoRe's scope |
| `ngx_http_geoip_module` | `geoip_*` | GeoIP lookup |
| `ngx_http_perl_module` | `perl_*` | Embedded Perl |
| `ngx_http_image_filter` | `image_filter` | Image processing |
| `ngx_http_flv_module` | `flv` | FLV streaming |
| `ngx_http_mp4_module` | `mp4` | MP4 streaming |
| `ngx_http_dav_module` | `dav_*` | WebDAV |
| `ngx_http_xslt_module` | `xslt_*` | XSLT transforms |
| `ngx_http_slice_module` | `slice` | Range request slicing |
| `ngx_http_random_index` | `random_index` | Random file selection |
| `ngx_http_secure_link` | `secure_link` | Secure link authentication |
| `ngx_http_geo_module` | `geo` | IP-to-variable mapping |
| `ngx_http_split_clients` | Already in GoRe as `split_clients` | — |

---

## Summary

| Category | Total Directives | Implemented | Partial | Stub | Not Implemented | Out of Scope |
|----------|-----------------|-------------|---------|------|----------------|--------------|
| Core | 45 | 21 | 4 | 0 | 7 | 6 |
| Proxy | 40 | 16 | 3 | 0 | 21 | 0 |
| SSL | 16 | 2 | 2 | 0 | 14 | 3 |
| Compression | 12 | 11 | 0 | 0 | 1 | 0 |
| Access | 2 | 2 | 0 | 0 | 0 | 0 |
| Rate Limit | 4 | 3 | 0 | 1 | 0 | 0 |
| Conn Limit | 4 | 3 | 0 | 1 | 0 | 0 |
| Logging | 6 | 2 | 1 | 0 | 3 | 0 |
| Headers | 3 | 3 | 0 | 0 | 0 | 0 |
| Rewrite | 5 | 2 | 0 | 0 | 3 | 0 |
| Auth Basic | 3 | 2 | 0 | 0 | 1 | 0 |
| Auth Request | 2 | 1 | 0 | 0 | 1 | 0 |
| Sub Filter | 3 | 3 | 0 | 0 | 0 | 0 |
| Mirror | 2 | 2 | 0 | 0 | 0 | 0 |
| Real IP | 3 | 3 | 0 | 0 | 0 | 0 |
| Upstream | 11 | 11 | 0 | 0 | 3 | 0 |
| SSL Session | 2 | 0 | 0 | 0 | 0 | 2 |
| **Total** | **167** | **78** | **11** | **2** | **65** | **14** |

**Phase 1 cleared 30 gaps** (48 → 78 implemented, 29% → 47% coverage).

### Top Priority Gaps (High frequency in production nginx configs)

| # | Directive | Module | Why Needed | Effort |
|---|-----------|--------|-----------|--------|
| 1 | `error_page` | core | ✅ Implemented | — |
| 2 | `proxy_connect_timeout` | proxy | ✅ Already implemented | — |
| 3 | `proxy_read_timeout` | proxy | ✅ Already implemented | — |
| 4 | `proxy_send_timeout` | proxy | ✅ Already implemented | — |
| 5 | `proxy_next_upstream` flags | proxy | ✅ Implemented | — |
| 6 | `proxy_redirect` | proxy | ✅ Implemented | — |
| 7 | `proxy_buffer_size` | proxy | ✅ Implemented | — |
| 8 | `proxy_request_buffering` | proxy | Buffer client body before proxying | Medium |
| 9 | `proxy_intercept_errors` | proxy | Custom error pages from upstream | Medium |
| 10 | `keepalive_timeout` (configurable) | core | ✅ Implemented | — |
| 11 | `keepalive_requests` | core | ✅ Implemented | — |
| 12 | `ssl_session_cache` | ssl | ✅ Go stdlib handles internally | — |
| 13 | `ssl_prefer_server_ciphers` | ssl | ✅ Go stdlib handles internally | — |
| 14 | `server_tokens` | core | ✅ Implemented | — |
| 15 | `gzip_min_length` | gzip | ✅ Implemented | — |
| 16 | `gzip_vary` | gzip | ✅ Implemented | — |
| 17 | `real_ip_recursive` | realip | ✅ Implemented | — |
| 18 | `limit_req_status` | ratelimit | ✅ Implemented | — |
| 19 | `auth_request_set` | auth_request | Pass auth response headers to backend | Medium |
| 20 | `sub_filter_once` | sub_filter | ✅ Implemented | — |

**Remaining Phase 2/3 gaps:** `proxy_request_buffering`, `proxy_intercept_errors`, `auth_request_set`, `proxy_ssl_*`, `proxy_cache_*`, `ssl_stapling`, `set` (variables), `if` (conditionals), `alias`, `slow_start`, `resolve`, `zone`.
