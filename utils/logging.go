package utils

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggingOptions struct {
	File string // empty disables file logger

	Level slog.Level // default info

	MaxSizeMB  int // default 25
	MaxBackups int // default 5
	MaxAgeDays int // default 14
	Compress   bool // default true

	AppName    string
	AppVersion string
}

type Logger struct {
	L     *slog.Logger
	closer *lumberjack.Logger
}

func New(opts LoggingOptions) (*Logger, error) {
	if strings.TrimSpace(opts.File) == "" {
		// No file logging; caller can use slog.Default() or stderr handler separately.
		return &Logger{L: nil, closer: nil}, nil
	}

	if err := EnsureDir(opts.File); err != nil {
		return nil, err
	}

	lj := &lumberjack.Logger{
		Filename:   opts.File,
		MaxSize:    defInt(opts.MaxSizeMB, 25),
		MaxBackups: defInt(opts.MaxBackups, 5),
		MaxAge:     defInt(opts.MaxAgeDays, 14),
		Compress:   defBool(opts.Compress, true),
	}

	level := opts.Level
	if level == 0 {
		level = slog.LevelInfo
	}

	h := slog.NewJSONHandler(lj, &slog.HandlerOptions{Level: level})
	l := slog.New(h)

	if opts.AppName != "" {
		l = l.With(slog.String("app", opts.AppName))
	}
	if opts.AppVersion != "" {
		l = l.With(slog.String("version", opts.AppVersion))
	}

	return &Logger{L: l, closer: lj}, nil
}

func (lg *Logger) Close() error {
	if lg == nil || lg.closer == nil {
		return nil
	}
	return lg.closer.Close()
}

func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func defInt(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}

// Nota: bool default. Si quieres soportar false explícito, pásalo siempre.
func defBool(v, d bool) bool {
	if v == false && d == true {
		return d
	}
	return v
}