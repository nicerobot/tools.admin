package app

import (
	"io"
	"log/slog"
)

// handlerFunc constructs a slog.Handler for a given writer and options.
type handlerFunc func(io.Writer, *slog.HandlerOptions) slog.Handler

// handlerType names a supported log output format.
type handlerType string

const (
	handlerTypeJSON handlerType = "json"
	handlerTypeText handlerType = "text"
)

// handlers maps each handler type to its constructor.
var handlers = map[handlerType]handlerFunc{
	handlerTypeText: func(w io.Writer, opts *slog.HandlerOptions) slog.Handler { return slog.NewTextHandler(w, opts) },
	handlerTypeJSON: func(w io.Writer, opts *slog.HandlerOptions) slog.Handler { return slog.NewJSONHandler(w, opts) },
}

// handler builds the slog.Handler for this handler type.
func (h handlerType) handler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return handlers[h](w, opts)
}

// handlerTypeFor selects a handler type from a log format, defaulting to text.
func handlerTypeFor(format LogFormat) handlerType {
	if format == LogFormat(handlerTypeJSON) {
		return handlerTypeJSON
	}
	return handlerTypeText
}
