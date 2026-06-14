package config

import (
	"os"

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
	Addr string `yaml:"addr"`
	TLS  *TLS   `yaml:"tls,omitempty"`
	HTTP2 *HTTP2 `yaml:"http2,omitempty"`
}

type TLS struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
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

type HTTPConfig struct {
	Servers []Server `yaml:"server"`
}

type Server struct {
	Name      string     `yaml:"name"`
	Locations []Location `yaml:"locations"`
}

type Location struct {
	Path      string  `yaml:"path"`
	Root      string  `yaml:"root,omitempty"`
	Proxy     *Proxy  `yaml:"proxy,omitempty"`
	Return    string  `yaml:"return,omitempty"`
	Autoindex *bool   `yaml:"autoindex,omitempty"`
}

type Proxy struct {
	Upstream   string `yaml:"upstream"`
	BufferSize string `yaml:"buffer_size,omitempty"`
}

type Upstream struct {
	Strategy  string           `yaml:"strategy"`
	Servers   []UpstreamServer `yaml:"servers"`
	Keepalive int              `yaml:"keepalive,omitempty"`
}

type UpstreamServer struct {
	Addr   string `yaml:"addr"`
	Weight int    `yaml:"weight,omitempty"`
}

type ModulesConfig struct {
	Gzip      *GzipConfig      `yaml:"gzip,omitempty"`
	Access    *AccessConfig    `yaml:"access,omitempty"`
	RateLimit *RateLimitConfig `yaml:"rate_limit,omitempty"`
	Headers   *HeadersConfig   `yaml:"headers,omitempty"`
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
