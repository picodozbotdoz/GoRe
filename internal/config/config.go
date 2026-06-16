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
	Addr  string `yaml:"addr"`
	TLS   *TLS   `yaml:"tls,omitempty"`
	HTTP2 *HTTP2 `yaml:"http2,omitempty"`
	HTTP3 *HTTP3 `yaml:"http3,omitempty"`
}

type TLS struct {
	Cert       string   `yaml:"cert"`
	Key        string   `yaml:"key"`
	Ciphers    []string `yaml:"ciphers,omitempty"`
	MinVersion string   `yaml:"min_version,omitempty"`
}

type HTTP2 struct {
	Enabled             *bool `yaml:"enabled,omitempty"`
	MaxConcurrentStreams int   `yaml:"max_concurrent_streams,omitempty"`
	MaxFrameSize        int   `yaml:"max_frame_size,omitempty"`
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
	Enabled    *bool `yaml:"enabled,omitempty"`
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
	Path         string            `yaml:"path"`
	Root         string            `yaml:"root,omitempty"`
	Proxy        *Proxy            `yaml:"proxy,omitempty"`
	Return       string            `yaml:"return,omitempty"`
	Rewrite      *Rewrite          `yaml:"rewrite,omitempty"`
	Autoindex    *bool             `yaml:"autoindex,omitempty"`
	CacheControl string            `yaml:"cache_control,omitempty"`
	TryFiles     []string          `yaml:"try_files,omitempty"`
	AuthRequest  string            `yaml:"auth_request,omitempty"`
	SubFilter    map[string]string `yaml:"sub_filter,omitempty"`
}

type Rewrite struct {
	Pattern     string `yaml:"pattern"`
	Replacement string `yaml:"replacement"`
	Code        int    `yaml:"code,omitempty"`
}

type Proxy struct {
	Upstream   string `yaml:"upstream"`
	Buffering  *bool  `yaml:"buffering,omitempty"`
	BufferSize string `yaml:"buffer_size,omitempty"`
}

type Upstream struct {
	Strategy        string            `yaml:"strategy"`
	Servers         []UpstreamServer  `yaml:"servers"`
	SetHeaders      map[string]string `yaml:"set_headers,omitempty"`
	Buffering       *bool             `yaml:"buffering,omitempty"`
	Retries         int               `yaml:"retries,omitempty"`
	HealthCheck     *HealthCheckConfig `yaml:"health_check,omitempty"`
	Keepalive       int               `yaml:"keepalive,omitempty"`
	ConnectTimeout  int               `yaml:"connect_timeout,omitempty"`
	ReadTimeout     int               `yaml:"read_timeout,omitempty"`
	SendTimeout     int               `yaml:"send_timeout,omitempty"`
	IdleTimeout     int               `yaml:"idle_timeout,omitempty"`
}

type HealthCheckConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval int    `yaml:"interval,omitempty"`
	Path     string `yaml:"path,omitempty"`
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
}

type ModulesConfig struct {
	Gzip              *GzipConfig      `yaml:"gzip,omitempty"`
	Brotli            *BrotliConfig    `yaml:"brotli,omitempty"`
	Access            *AccessConfig    `yaml:"access,omitempty"`
	RateLimit         *RateLimitConfig `yaml:"rate_limit,omitempty"`
	LimitConn         *LimitConnConfig `yaml:"limit_conn,omitempty"`
	Headers           *HeadersConfig   `yaml:"headers,omitempty"`
	AccessLog         *AccessLogConfig `yaml:"access_log,omitempty"`
	ErrorLog          *ErrorLogConfig  `yaml:"error_log,omitempty"`
	ClientMaxBodySize string           `yaml:"client_max_body_size,omitempty"`
	Status            *StatusConfig    `yaml:"status,omitempty"`
	RealIP            *RealIPConfig    `yaml:"real_ip,omitempty"`
	BasicAuth         *BasicAuthConfig `yaml:"basic_auth,omitempty"`
}

type BrotliConfig struct {
	Enabled bool     `yaml:"enabled"`
	Level   int      `yaml:"level,omitempty"`
	Types   []string `yaml:"types,omitempty"`
}

type BasicAuthConfig struct {
	Realm    string            `yaml:"realm,omitempty"`
	Users    map[string]string `yaml:"users,omitempty"`
}

type RealIPConfig struct {
	From string `yaml:"from,omitempty"`
}

type LimitConnConfig struct {
	Zone        string `yaml:"zone,omitempty"`
	Connections int    `yaml:"connections"`
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
	Zone  string `yaml:"zone"`
	Rate  string `yaml:"rate"`
	Burst int    `yaml:"burst,omitempty"`
}

type HeadersConfig struct {
	Add    map[string]string `yaml:"add,omitempty"`
	Remove []string          `yaml:"remove,omitempty"`
}

type AccessLogConfig struct {
	Enabled bool   `yaml:"enabled"`
	Output  string `yaml:"output,omitempty"`
	Format  string `yaml:"format,omitempty"`
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
