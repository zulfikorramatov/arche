package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"go.elastic.co/apm/module/apmslog/v2"
)

type Config struct {
	Level string
}

type Logger struct {
	*slog.Logger
}

func New(cfg Config) (*Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				a.Key = "level_name"
				a.Value = slog.StringValue(a.Value.Any().(slog.Level).String())
			case slog.TimeKey:
				a.Key = "@timestamp"
			}
			return a
		},
	})

	apmHandler := apmslog.NewApmHandler(apmslog.WithHandler(h))

	return &Logger{Logger: slog.New(apmHandler)}, nil
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

func (l *Logger) Info(msg string, args ...any) {
	args = append(args, getSource(2))
	l.Logger.Info(msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	args = append(args, getSource(2))
	l.Logger.Debug(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	args = append(args, getSource(2))
	l.Logger.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	args = append(args, getSource(2))
	l.Logger.Error(msg, args...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	args = append(args, getSource(2))
	l.Logger.InfoContext(ctx, msg, args...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	args = append(args, getSource(2))
	l.Logger.DebugContext(ctx, msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	args = append(args, getSource(2))
	l.Logger.WarnContext(ctx, msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	args = append(args, getSource(2))
	l.Logger.ErrorContext(ctx, msg, args...)
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", s)
	}
}

func getSource(skip int) slog.Attr {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return slog.Attr{}
	}

	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
	}

	return slog.Group("source",
		"function", funcName,
		"file", file,
		"line", line,
	)
}
