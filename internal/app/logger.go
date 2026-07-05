package app

import (
	"io"
	"log/slog"

	"github.com/urfave/cli/v3"
)

// LoggerMetadataKey is the cli.Command metadata key under which the configured
// logger is stored so every command resolves the same instance.
const LoggerMetadataKey = "logger"

type (
	// LogFormat selects the log output format ("text" or "json").
	LogFormat string
	// LogLevel selects the minimum log level ("debug", "info", "warn", "error").
	LogLevel string
)

// LoggerConfig holds the logging configuration bound from the root command flags.
type LoggerConfig struct {
	LogLevel  LogLevel
	LogFormat LogFormat
}

// GetLoggerFunc resolves the logger for a command. It is the seam tests use to
// inject a logger and production uses to build one from LoggerConfig.
type GetLoggerFunc func(*cli.Command) *slog.Logger

// parseLevel converts a LogLevel to a slog.Level, defaulting to info when the
// value is empty or unrecognized.
func parseLevel(level LogLevel) slog.Level {
	parsed := slog.LevelInfo
	if err := parsed.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return parsed
}

// NewLogger builds a logger from cfg, writing to w. This is the single place
// that turns logging configuration into a slog.Logger. It returns a value —
// slog.Logger is a small, copyable handle around its handler.
func NewLogger(w io.Writer, cfg LoggerConfig) slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}
	return *slog.New(handlerTypeFor(cfg.LogFormat).handler(w, opts))
}

// GetLogger retrieves the configured logger from command metadata, falling back
// to the slog default when no command or logger is available.
func GetLogger(command *cli.Command) *slog.Logger {
	if command != nil {
		if logger, ok := command.Root().Metadata[LoggerMetadataKey].(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}

var _ GetLoggerFunc = GetLogger
