package logger

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(opts ...Option) *zap.Logger {
	o := &option{
		level:   zapcore.InfoLevel,
		encoder: NewConsoleEncoder,
		writer:  os.Stdout,
	}
	for _, opt := range opts {
		opt(o)
	}

	core := zapcore.NewCore(
		o.encoder(newEncoderConfig()),
		zap.CombineWriteSyncers(zapcore.AddSync(o.writer)),
		o.level,
	).With(o.fields) // 自带node 信息
	// 大于error增加堆栈信息
	return zap.New(core).WithOptions(zap.AddCaller(), zap.AddCallerSkip(o.skip),
		zap.AddStacktrace(zapcore.DPanicLevel))
}

func newEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:     "Message",
		LevelKey:       "Level",
		TimeKey:        "Time",
		NameKey:        "Logger",
		CallerKey:      "Caller",
		StacktraceKey:  "Stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
}

type option struct {
	skip    int
	level   zapcore.Level
	encoder func(zapcore.EncoderConfig) zapcore.Encoder
	writer  io.Writer
	fields  []zap.Field
}

type Option func(*option)

func WithSkip(skip int) Option {
	return func(o *option) {
		o.skip = skip
	}
}

func WithLevel(level zapcore.Level) Option {
	return func(o *option) {
		o.level = level
	}
}

func WithEncoder(encoder func(zapcore.EncoderConfig) zapcore.Encoder) Option {
	return func(o *option) {
		o.encoder = encoder
	}
}

func WithWriter(w io.Writer) Option {
	return func(o *option) {
		o.writer = w
	}
}

func WithFields(fields ...zap.Field) Option {
	return func(o *option) {
		o.fields = fields
	}
}
