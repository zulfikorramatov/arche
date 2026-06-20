package logger

import (
	"fmt"

	"go.elastic.co/apm/module/apmzap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	FormatJSON    = "json"
	FormatConsole = "console"
)

type Config struct {
	Level        string
	Format       string
	APM          bool
	SamplingRate int
}

func New(cfg Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse level %q: %w", cfg.Level, err)
	}

	switch cfg.Format {
	case FormatJSON, FormatConsole:
		// ok
	default:
		return nil, fmt.Errorf("unknown format %q: want %q or %q", cfg.Format, FormatJSON, FormatConsole)
	}

	var sampling *zap.SamplingConfig
	if cfg.SamplingRate > 0 {
		sampling = &zap.SamplingConfig{
			Initial:    cfg.SamplingRate,
			Thereafter: cfg.SamplingRate,
		}
	}

	zapCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      false,
		Sampling:         sampling,
		Encoding:         cfg.Format,
		EncoderConfig:    encoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	var opts []zap.Option
	if cfg.APM {
		opts = append(opts, zap.WrapCore((&apmzap.Core{}).WrapCore))
	}

	log, err := zapCfg.Build(opts...)
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return log, nil
}

func encoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "severity",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}
