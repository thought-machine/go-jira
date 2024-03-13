package jira

import (
	"context"
	"io"
	"log/slog"
	"os"
)

const (
	levelTrace   = slog.Level(-8)
	levelDebug   = slog.LevelDebug
	levelInfo    = slog.LevelInfo
	levelWarning = slog.LevelWarn
	levelError   = slog.LevelError
	levelPanic   = slog.Level(12)
	levelFatal   = slog.Level(16)

	// logLevelEnvKey is the name of the environment variable whose value defines the logging level of
	// the logger used by the default client.
	logLevelEnvKey = "VERBOSITY"
)

// logger is the logger used by the default client. It writes log messages to stderr.
var logger = slog.New(newRetryablehttpHandler(os.Stderr))

// retryablehttpHandler handles log messages produced for LeveledLoggers by go-retryablehttp. It
// wraps another handler, mutating the log records it receives before forwarding them on to the
// wrapped handler.
type retryablehttpHandler struct {
	// handler is the wrapped handler.
	handler slog.Handler

	// level is the minimum level for which messages should be logged.
	level slog.Level
}

// newRetryablehttpHandler creates a retryablehttpHandler whose minimum logging level is the value
// of the VERBOSITY environment variable.
func newRetryablehttpHandler(output io.Writer) *retryablehttpHandler {
	return &retryablehttpHandler{
		handler: slog.NewTextHandler(output, nil),
		level:   defaultLogLevel(),
	}
}

// Enabled reports whether the handler handles records at the given level. The handler ignores
// records whose level is lower.
//
// Enabled is called before Handle for performance reasons. Because retryablehttpHandler changes the
// level of some messages in Handle, records cannot be rejected at this point solely because of
// their level, so Enabled always returns true. However, Handle will only forward them on to the
// wrapped handler if their level is at least h.level.
func (h *retryablehttpHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// WithAttrs returns a new retryablehttpHandler whose attributes consists of h's attributes followed
// by attrs.
func (h *retryablehttpHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &retryablehttpHandler{
		handler: h.handler.WithAttrs(attrs),
		level:   h.level,
	}
}

// WithGroup returns a new retryablehttpHandler with the given group appended to the receiver's
// existing groups.
func (h *retryablehttpHandler) WithGroup(name string) slog.Handler {
	return &retryablehttpHandler{
		handler: h.handler.WithGroup(name),
		level:   h.level,
	}
}

// Handle processes a record created by a retryablehttp.Client and potentially forwards it on to the
// wrapped handler.
//
// Handle mutates records as follows:
// - records with a "retrying request" message (which are created by retryablehttp.Client when a
//   request fails due to a server-side error or a retryable client-side error, e.g. when the Jira
//   API's rate limits have been exceeded) are increased from debug level to warning level, to make
//   them more visible.
func (h *retryablehttpHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Message == "retrying request" {
		r = r.Clone()
		r.Level = levelWarning
	}
	if r.Level < h.level {
		return nil
	}
	return h.handler.Handle(ctx, r)
}

func defaultLogLevel() slog.Level {
	switch os.Getenv(logLevelEnvKey) {
	case "TRACE":
		return levelTrace
	case "DEBUG":
		return levelDebug
	case "INFO":
		return levelInfo
	case "WARNING":
		return levelWarning
	case "ERROR":
		return levelError
	case "PANIC":
		return levelPanic
	case "FATAL":
		return levelFatal
	default:
		return levelInfo
	}
}
