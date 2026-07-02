package logger

import (
	"fmt"
	"log/slog"
	"os"
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
