package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

type Format string

const (
	FormatPlain Format = "plain"
	FormatJSON  Format = "json"
)

var (
	defaultLogger *slog.Logger
	currentLevel  = LevelInfo
	osExit        = os.Exit // For testing
)

func init() {
	SetFormat(FormatPlain)
	SetLevel(LevelInfo)
}

// SetLevel sets the current logging level
func SetLevel(level Level) {
	currentLevel = level
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if defaultLogger != nil {
		// Preserve existing format
		if _, ok := defaultLogger.Handler().(*slog.JSONHandler); ok {
			handler = slog.NewJSONHandler(os.Stderr, opts)
		} else {
			handler = newPlainHandler(os.Stderr, opts)
		}
	} else {
		handler = newPlainHandler(os.Stderr, opts)
	}

	defaultLogger = slog.New(handler)
}

// SetFormat sets the output format
func SetFormat(format Format) error {
	opts := &slog.HandlerOptions{
		Level: currentLevel,
	}

	var handler slog.Handler
	switch format {
	case FormatJSON:
		handler = slog.NewJSONHandler(os.Stderr, opts)
	case FormatPlain:
		handler = newPlainHandler(os.Stderr, opts)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	defaultLogger = slog.New(handler)
	return nil
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	defaultLogger.Debug(msg)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	defaultLogger.Info(msg)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	defaultLogger.Warn(msg)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	defaultLogger.Error(msg)
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	defaultLogger.Error(msg)
	osExit(1)
}

// ParseLevel parses a string level into a Level value
func ParseLevel(level string) (Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
}

// ParseFormat parses a string format into a Format value
func ParseFormat(format string) (Format, error) {
	switch Format(format) {
	case FormatPlain:
		return FormatPlain, nil
	case FormatJSON:
		return FormatJSON, nil
	default:
		return FormatPlain, fmt.Errorf("invalid format: %s", format)
	}
}

// plainHandler implements a custom handler for plain text format
type plainHandler struct {
	opts   slog.HandlerOptions
	w      io.Writer
	attrs  []slog.Attr
	groups []string
}

func newPlainHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &plainHandler{
		opts:   *opts,
		w:      w,
		attrs:  make([]slog.Attr, 0),
		groups: make([]string, 0),
	}
}

func (h *plainHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *plainHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()
	if r.Level == LevelInfo {
		level = "" // Don't show level for info messages
	} else {
		level = fmt.Sprintf("[%s] ", strings.ToUpper(level))
	}

	msg := fmt.Sprintf("%s%s\n", level, r.Message)
	_, err := io.WriteString(h.w, msg)
	return err
}

func (h *plainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &plainHandler{
		opts:   h.opts,
		w:      h.w,
		attrs:  append(h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *plainHandler) WithGroup(name string) slog.Handler {
	return &plainHandler{
		opts:   h.opts,
		w:      h.w,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}
