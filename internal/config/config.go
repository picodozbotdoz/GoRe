package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WorkerProcesses string              `yaml:"worker_processes"`
	Listen          []Listen            `yaml:"listen"`
	HTTP            HTTPConfig          `yaml:"http"`
	Upstreams       map[string]Upstream `yaml:"upstreams"`
	Modules         ModulesConfig       `yaml:"modules"`
}

type Listen struct {
	Addr              string `yaml:"addr"`
	TLS               *TLS   `yaml:"tls,omitempty"`
	HTTP2             *HTTP2 `yaml:"http2,omitempty"`
	HTTP3             *HTTP3 `yaml:"http3,omitempty"`
	KeepAliveTimeout  int    `yaml:"keepalive_timeout,omitempty"`
	KeepAliveRequests int    `yaml:"keepalive_requests,omitempty"`
}

type TLS struct {
	Cert              string   `yaml:"cert"`
	Key               string   `yaml:"key"`
	Ciphers           []string `yaml:"ciphers,omitempty"`
	MinVersion        string   `yaml:"min_version,omitempty"`
	SessionTimeout    int      `yaml:"session_timeout,omitempty"`
	ClientCertificate string   `yaml:"client_certificate,omitempty"`
	VerifyClient      bool     `yaml:"verify_client,omitempty"`
	VerifyDepth       int      `yaml:"verify_depth,omitempty"`
	RejectHandshake   bool     `yaml:"reject_handshake,omitempty"`
}

func (t *TLS) GetSessionTimeout() int {
	if t == nil || t.SessionTimeout == 0 {
		return 300
	}
	return t.SessionTimeout
}

func (t *TLS) GetVerifyDepth() int {
	if t == nil || t.VerifyDepth == 0 {
		return 1
	}
	return t.VerifyDepth
}

type HTTP2 struct {
	Enabled              *bool `yaml:"enabled,omitempty"`
	MaxConcurrentStreams int   `yaml:"max_concurrent_streams,omitempty"`
	MaxFrameSize         int   `yaml:"max_frame_size,omitempty"`
}

func (h *HTTP2) GetMaxConcurrentStreams() int {
	if h == nil || h.MaxConcurrentStreams == 0 {
		return 250
	}
	return h.MaxConcurrentStreams
}

func (h *HTTP2) GetMaxFrameSize() int {
	if h == nil || h.MaxFrameSize == 0 {
		return 1048576
	}
	return h.MaxFrameSize
}

type HTTP3 struct {
	Enabled     *bool `yaml:"enabled,omitempty"`
	MaxStreams  int   `yaml:"max_streams,omitempty"`
	IdleTimeout int   `yaml:"idle_timeout,omitempty"`
}

func (h *HTTP3) GetMaxStreams() int {
	if h == nil || h.MaxStreams == 0 {
		return 100
	}
	return h.MaxStreams
}

func (h *HTTP3) GetIdleTimeout() int {
	if h == nil || h.IdleTimeout == 0 {
		return 30
	}
	return h.IdleTimeout
}

type HTTPConfig struct {
	Servers []Server `yaml:"server"`
}

type Server struct {
	Name      string     `yaml:"name"`
	Locations []Location `yaml:"locations"`
}

type Location struct {
	Path           string            `yaml:"path"`
	Root           string            `yaml:"root,omitempty"`
	Alias          string            `yaml:"alias,omitempty"`
	Proxy          *Proxy            `yaml:"proxy,omitempty"`
	Return         string            `yaml:"return,omitempty"`
	Rewrite        *Rewrite          `yaml:"rewrite,omitempty"`
	Autoindex      *bool             `yaml:"autoindex,omitempty"`
	CacheControl   string            `yaml:"cache_control,omitempty"`
	TryFiles       []string          `yaml:"try_files,omitempty"`
	AuthRequest    string            `yaml:"auth_request,omitempty"`
	AuthRequestSet map[string]string `yaml:"auth_request_set,omitempty"`
	LimitExcept    []string          `yaml:"limit_except,omitempty"`
	Internal       bool              `yaml:"internal,omitempty"`
	Satisfy        string            `yaml:"satisfy,omitempty"`
	SubFilter      map[string]string `yaml:"sub_filter,omitempty"`
	SubFilterOnce  *bool             `yaml:"sub_filter_once,omitempty"`
	SubFilterTypes []string          `yaml:"sub_filter_types,omitempty"`
	Mirror         string            `yaml:"mirror,omitempty"`
}

type Rewrite struct {
	Pattern     string `yaml:"pattern"`
	Replacement string `yaml:"replacement"`
	Code        int    `yaml:"code,omitempty"`
	Log         bool   `yaml:"log,omitempty"`
	Break       bool   `yaml:"break,omitempty"`
}

type Proxy struct {
	Upstream           string            `yaml:"upstream"`
	Buffering          *bool             `yaml:"buffering,omitempty"`
	BufferSize         string            `yaml:"buffer_size,omitempty"`
	RequestBuffering   *bool             `yaml:"request_buffering,omitempty"`
	InterceptErrors    bool              `yaml:"intercept_errors,omitempty"`
	ErrorPages         map[int]string    `yaml:"error_pages,omitempty"`
	CookieDomain       string            `yaml:"cookie_domain,omitempty"`
	CookiePath         string            `yaml:"cookie_path,omitempty"`
	Method             string            `yaml:"method,omitempty"`
	Buffers            string            `yaml:"buffers,omitempty"`
	BusyBuffersSize    string            `yaml:"busy_buffers_size,omitempty"`
	Redirect           string            `yaml:"redirect,omitempty"`
	PassRequestHeaders *bool             `yaml:"pass_request_headers,omitempty"`
	PassRequestBody    *bool             `yaml:"pass_request_body,omitempty"`
	MaxTempFileSize    string            `yaml:"max_temp_file_size,omitempty"`
}

type Upstream struct {
	Strategy            string             `yaml:"strategy"`
	Servers             []UpstreamServer   `yaml:"servers"`
	SetHeaders          map[string]string  `yaml:"set_headers,omitempty"`
	Buffering           *bool              `yaml:"buffering,omitempty"`
	Retries             int                `yaml:"retries,omitempty"`
	HealthCheck         *HealthCheckConfig `yaml:"health_check,omitempty"`
	Cache               *CacheConfig       `yaml:"cache,omitempty"`
	ProxySSL            *ProxySSL          `yaml:"proxy_ssl,omitempty"`
	ProxyProtocol       bool               `yaml:"proxy_protocol,omitempty"`
	HideHeaders         []string           `yaml:"hide_headers,omitempty"`
	SocketKeepalive     *bool              `yaml:"socket_keepalive,omitempty"`
	MaxTempFileSize     string             `yaml:"max_temp_file_size,omitempty"`
	Keepalive           int                `yaml:"keepalive,omitempty"`
	KeepaliveTimeout    int                `yaml:"keepalive_timeout,omitempty"`
	KeepaliveRequests   int                `yaml:"keepalive_requests,omitempty"`
	ConnectTimeout      int                `yaml:"connect_timeout,omitempty"`
	ReadTimeout         int                `yaml:"read_timeout,omitempty"`
	SendTimeout         int                `yaml:"send_timeout,omitempty"`
	IdleTimeout         int                `yaml:"idle_timeout,omitempty"`
	NextUpstream        string             `yaml:"next_upstream,omitempty"`
	NextUpstreamTries   int                `yaml:"next_upstream_tries,omitempty"`
	NextUpstreamTimeout int                `yaml:"next_upstream_timeout,omitempty"`
}

type ProxySSL struct {
	Verify             bool   `yaml:"verify,omitempty"`
	Certificate        string `yaml:"certificate,omitempty"`
	CertificateKey     string `yaml:"certificate_key,omitempty"`
	TrustedCertificate string `yaml:"trusted_certificate,omitempty"`
	Protocols          string `yaml:"protocols,omitempty"`
	Ciphers            string `yaml:"ciphers,omitempty"`
	ServerName         string `yaml:"server_name,omitempty"`
	SessionReuse       *bool  `yaml:"session_reuse,omitempty"`
	Name               string `yaml:"name,omitempty"`
}

type HealthCheckConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval int    `yaml:"interval,omitempty"`
	Path     string `yaml:"path,omitempty"`
}

type CacheConfig struct {
	Enabled bool `yaml:"enabled"`
	TTL     int  `yaml:"ttl,omitempty"`
	MaxSize int  `yaml:"max_size,omitempty"`
}

func (h *HealthCheckConfig) GetInterval() int {
	if h == nil || h.Interval == 0 {
		return 10
	}
	return h.Interval
}

func (u *Upstream) GetRetries() int {
	if u == nil || u.Retries < 0 {
		return 0
	}
	return u.Retries
}

func (u *Upstream) GetBuffering() bool {
	if u == nil || u.Buffering == nil {
		return true
	}
	return *u.Buffering
}

func (u *Upstream) GetConnectTimeout() int {
	if u == nil || u.ConnectTimeout == 0 {
		return 60
	}
	return u.ConnectTimeout
}

func (u *Upstream) GetReadTimeout() int {
	if u == nil || u.ReadTimeout == 0 {
		return 60
	}
	return u.ReadTimeout
}

func (u *Upstream) GetSendTimeout() int {
	if u == nil || u.SendTimeout == 0 {
		return 60
	}
	return u.SendTimeout
}

func (u *Upstream) GetIdleTimeout() int {
	if u == nil || u.IdleTimeout == 0 {
		return 90
	}
	return u.IdleTimeout
}

type UpstreamServer struct {
	Addr   string `yaml:"addr"`
	Weight int    `yaml:"weight,omitempty"`
	Backup bool   `yaml:"backup,omitempty"`
	Down   bool   `yaml:"down,omitempty"`
}

type ModulesConfig struct {
	Gzip                *GzipConfig      `yaml:"gzip,omitempty"`
	Brotli              *BrotliConfig    `yaml:"brotli,omitempty"`
	Access              *AccessConfig    `yaml:"access,omitempty"`
	RateLimit           *RateLimitConfig `yaml:"rate_limit,omitempty"`
	LimitConn           *LimitConnConfig `yaml:"limit_conn,omitempty"`
	Headers             *HeadersConfig   `yaml:"headers,omitempty"`
	AccessLog           *AccessLogConfig `yaml:"access_log,omitempty"`
	ErrorLog            *ErrorLogConfig  `yaml:"error_log,omitempty"`
	ClientMaxBodySize   string           `yaml:"client_max_body_size,omitempty"`
	Status              *StatusConfig    `yaml:"status,omitempty"`
	RealIP              *RealIPConfig    `yaml:"real_ip,omitempty"`
	BasicAuth           *BasicAuthConfig `yaml:"basic_auth,omitempty"`
	Map                 []MapConfig      `yaml:"map,omitempty"`
	SplitClients        []SplitConfig    `yaml:"split_clients,omitempty"`
	Gunzip              *bool            `yaml:"gunzip,omitempty"`
	ErrorPage           *ErrorPageConfig `yaml:"error_page,omitempty"`
	ServerTokens        *bool            `yaml:"server_tokens,omitempty"`
	DefaultType         string           `yaml:"default_type,omitempty"`
	MergeSlashes        *bool            `yaml:"merge_slashes,omitempty"`
	ClientBodyTimeout   int              `yaml:"client_body_timeout,omitempty"`
	ClientHeaderTimeout int              `yaml:"client_header_timeout,omitempty"`
	SendTimeout         int              `yaml:"send_timeout,omitempty"`
	Resolver            string           `yaml:"resolver,omitempty"`
}

type SplitConfig struct {
	Source  string      `yaml:"source"`
	Target  string      `yaml:"target"`
	Rules   []SplitRule `yaml:"rules"`
	Default string      `yaml:"default"`
}

type SplitRule struct {
	Percent float64 `yaml:"percent"`
	Value   string  `yaml:"value"`
}

type MapConfig struct {
	Source  string    `yaml:"source"`
	Target  string    `yaml:"target"`
	Rules   []MapRule `yaml:"rules"`
	Default string    `yaml:"default,omitempty"`
}

type MapRule struct {
	Pattern string `yaml:"pattern"`
	Value   string `yaml:"value"`
}

type BasicAuthConfig struct {
	Realm string            `yaml:"realm,omitempty"`
	Users map[string]string `yaml:"users,omitempty"`
}

type ErrorPageConfig struct {
	Pages map[int]string `yaml:"pages,omitempty"`
}

type RealIPConfig struct {
	From      []string `yaml:"from,omitempty"`
	Recursive bool     `yaml:"recursive,omitempty"`
}

type LimitConnConfig struct {
	Zone        string `yaml:"zone,omitempty"`
	Connections int    `yaml:"connections"`
	LogLevel    string `yaml:"log_level,omitempty"`
}

type StatusConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path,omitempty"`
}

func (s *StatusConfig) GetPath() string {
	if s == nil || s.Path == "" {
		return "/status"
	}
	return s.Path
}

func ParseSize(s string) int64 {
	if s == "" {
		return 0
	}
	s = strings.TrimSpace(s)
	multiplier := int64(1)
	if idx := strings.IndexAny(s, "kKmMgG"); idx != -1 {
		numStr := s[:idx]
		suffix := strings.ToLower(s[idx:])
		num, _ := strconv.ParseInt(numStr, 10, 64)
		switch suffix {
		case "k", "kb":
			multiplier = 1024
		case "m", "mb":
			multiplier = 1024 * 1024
		case "g", "gb":
			multiplier = 1024 * 1024 * 1024
		}
		return num * multiplier
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

type GzipConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Level     int      `yaml:"level,omitempty"`
	Types     []string `yaml:"types,omitempty"`
	MinLength int      `yaml:"min_length,omitempty"`
	Vary      bool     `yaml:"vary,omitempty"`
	Proxied   bool     `yaml:"proxied,omitempty"`
	Disable   string   `yaml:"disable,omitempty"`
	Static    bool     `yaml:"static,omitempty"`
}

type BrotliConfig struct {
	Enabled bool     `yaml:"enabled"`
	Level   int      `yaml:"level,omitempty"`
	Types   []string `yaml:"types,omitempty"`
}

type AccessConfig struct {
	Rules []AccessRule `yaml:"rules"`
}

type AccessRule struct {
	Allow string `yaml:"allow,omitempty"`
	Deny  string `yaml:"deny,omitempty"`
}

type RateLimitConfig struct {
	Zone     string `yaml:"zone"`
	Rate     string `yaml:"rate"`
	Burst    int    `yaml:"burst,omitempty"`
	Status   int    `yaml:"status,omitempty"`
	LogLevel string `yaml:"log_level,omitempty"`
}

type HeadersConfig struct {
	Add     []HeaderEntry  `yaml:"add,omitempty"`
	Remove  []string       `yaml:"remove,omitempty"`
	Expires string         `yaml:"expires,omitempty"`
}

type HeaderEntry struct {
	Name   string `yaml:"name"`
	Value  string `yaml:"value"`
	Always bool   `yaml:"always,omitempty"`
}

type AccessLogConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Output     string `yaml:"output,omitempty"`
	Format     string `yaml:"format,omitempty"`
	Subrequest bool   `yaml:"subrequest,omitempty"`
}

func (c *AccessLogConfig) GetOutput() string {
	if c == nil || c.Output == "" {
		return "stdout"
	}
	return c.Output
}

func (c *AccessLogConfig) GetFormat() string {
	if c == nil || c.Format == "" {
		return ""
	}
	return c.Format
}

type ErrorLogConfig struct {
	Level  string `yaml:"level,omitempty"`
	Output string `yaml:"output,omitempty"`
}

func (c *ErrorLogConfig) GetLevel() string {
	if c == nil || c.Level == "" {
		return "info"
	}
	return c.Level
}

func (c *ErrorLogConfig) GetOutput() string {
	if c == nil || c.Output == "" {
		return "stderr"
	}
	return c.Output
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.WorkerProcesses == "" {
		cfg.WorkerProcesses = "auto"
	}
	if len(cfg.Listen) == 0 {
		cfg.Listen = []Listen{{Addr: ":80"}}
	}

	return &cfg, nil
}
