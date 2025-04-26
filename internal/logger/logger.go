package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Field = zapcore.Field

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	With(fields ...Field) Logger
	Sync() error
}

type logger struct {
	*zap.Logger
}

type Config struct {
	Level       string   `yaml:"level"`
	Encoding    string   `yaml:"encoding"`
	OutputPaths []string `yaml:"output_paths"`
	DevMode     bool     `yaml:"dev_mode"`
}

func New(cfg *Config) (Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	zapConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: cfg.DevMode,
		Encoding:    cfg.Encoding,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      cfg.OutputPaths,
		ErrorOutputPaths: []string{"stderr"},
	}

	l, err := zapConfig.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return &logger{Logger: l}, nil
}

func (l *logger) With(fields ...Field) Logger {
	return &logger{Logger: l.Logger.With(fields...)}
}

func String(key, value string) Field             { return zap.String(key, value) }
func Int(key string, value int) Field            { return zap.Int(key, value) }
func Int64(key string, value int64) Field        { return zap.Int64(key, value) }
func Bool(key string, value bool) Field          { return zap.Bool(key, value) }
func Float64(key string, value float64) Field    { return zap.Float64(key, value) }
func Error(key string, err error) Field          { return zap.Error(err) }
func Duration(key string, d time.Duration) Field { return zap.Duration(key, d) }

func (l *logger) Sync() error {
	return l.Logger.Sync()
}
