package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var defaultLogger *Logger

func init() {
	defaultLogger = &Logger{
		level:  LevelInfo,
		output: os.Stderr,
	}
	log.SetOutput(io.Discard)
}

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func parseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

type Logger struct {
	mu     sync.Mutex
	level  Level
	output io.Writer
}

type Config struct {
	Level     string           `yaml:"level,omitempty"`
	Output    string           `yaml:"output,omitempty"`
	AccessLog *AccessLogConfig `yaml:"access_log,omitempty"`
}

type AccessLogConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Output     string `yaml:"output,omitempty"`
	Format     string `yaml:"format,omitempty"`
	Subrequest bool   `yaml:"subrequest,omitempty"`
}

func Init(cfg *Config) {
	if cfg == nil {
		return
	}
	defaultLogger.SetLevel(parseLevel(cfg.Level))

	if cfg.Output != "" && cfg.Output != "stderr" && cfg.Output != "stdout" {
		f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "log: failed to open %s: %v\n", cfg.Output, err)
		} else {
			defaultLogger.SetOutput(f)
		}
	}
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

func (l *Logger) Level() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

func (l *Logger) log(level Level, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level < l.level {
		return
	}
	ts := time.Now().Format("2006/01/02 15:04:05")
	text := fmt.Sprintf(msg, args...)
	fmt.Fprintf(l.output, "%s [%s] %s\n", ts, level, text)
}

func (l *Logger) Debugf(msg string, args ...any) { l.log(LevelDebug, msg, args...) }
func (l *Logger) Infof(msg string, args ...any)  { l.log(LevelInfo, msg, args...) }
func (l *Logger) Warnf(msg string, args ...any)  { l.log(LevelWarn, msg, args...) }
func (l *Logger) Errorf(msg string, args ...any) { l.log(LevelError, msg, args...) }

func Debugf(msg string, args ...any) { defaultLogger.Debugf(msg, args...) }
func Infof(msg string, args ...any)  { defaultLogger.Infof(msg, args...) }
func Warnf(msg string, args ...any)  { defaultLogger.Warnf(msg, args...) }
func Errorf(msg string, args ...any) { defaultLogger.Errorf(msg, args...) }
