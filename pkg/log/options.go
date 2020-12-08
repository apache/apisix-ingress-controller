package log

import (
	"go.uber.org/zap/zapcore"
)

// Option configures how to set up logger.
type Option interface {
	apply(*options)
}

type funcOption struct {
	do func(*options)
}

func (fo *funcOption) apply(o *options) {
	fo.do(o)
}

type options struct {
	writeSyncer zapcore.WriteSyncer
	outputFile  string
	logLevel    string
	context     string
}

// WithLogLevel sets the log level.
func WithLogLevel(level string) Option {
	return &funcOption{
		do: func(o *options) {
			o.logLevel = level
		},
	}
}

// WithOutputFile sets the output file path.
func WithOutputFile(file string) Option {
	return &funcOption{
		do: func(o *options) {
			o.outputFile = file
		},
	}
}

// WithWriteSyncer is a low level API which sets the underlying
// WriteSyncer by providing a zapcore.WriterSyncer,
// which has high priority than WithOutputFile.
func WithWriteSyncer(ws zapcore.WriteSyncer) Option {
	return &funcOption{
		do: func(o *options) {
			o.writeSyncer = ws
		},
	}
}
