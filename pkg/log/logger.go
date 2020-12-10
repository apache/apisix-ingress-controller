 // Licensed to the Apache Software Foundation (ASF) under one or more
 // contributor license agreements.  See the NOTICE file distributed with
 // this work for additional information regarding copyright ownership.
 // The ASF licenses this file to You under the Apache License, Version 2.0
 // (the "License"); you may not use this file except in compliance with
 // the License.  You may obtain a copy of the License at
 //
 //     http://www.apache.org/licenses/LICENSE-2.0
 //
 // Unless required by applicable law or agreed to in writing, software
 // distributed under the License is distributed on an "AS IS" BASIS,
 // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 // See the License for the specific language governing permissions and
 // limitations under the License.
package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"go.uber.org/zap/zapcore"
)

var (
	levelMap = map[string]zapcore.Level{
		zapcore.DebugLevel.String(): zapcore.DebugLevel,
		zapcore.InfoLevel.String():  zapcore.InfoLevel,
		zapcore.WarnLevel.String():  zapcore.WarnLevel,
		zapcore.ErrorLevel.String(): zapcore.ErrorLevel,
		zapcore.PanicLevel.String(): zapcore.PanicLevel,
		zapcore.FatalLevel.String(): zapcore.FatalLevel,
	}
)

// Logger is a log object, which exposes standard APIs like
// errorf, error, warn, warnf and etcd.
type Logger struct {
	writer io.Writer
	core   zapcore.Core
	level  zapcore.Level
}

func (logger *Logger) write(level zapcore.Level, message string, fields []zapcore.Field) {
	e := zapcore.Entry{
		Level:   level,
		Time:    time.Now(),
		Message: message,
		Caller:  zapcore.NewEntryCaller(runtime.Caller(2)),
	}

	_ = logger.core.Write(e, fields)
}

// Sync flushes all buffered logs to the their destination.
func (logger *Logger) Sync() (err error) {
	file, ok := logger.writer.(*os.File)
	if ok && file != os.Stdout && file != os.Stderr {
		err = logger.core.Sync()
	}
	return
}

// Close flushes all buffered logs and closes the underlying writer.
func (logger *Logger) Close() (err error) {
	closer, ok := logger.writer.(io.Closer)
	if ok {
		return closer.Close()
	}
	return nil
}

// Debug uses the fmt.Sprint to construct and log a message.
func (logger *Logger) Debug(args ...interface{}) {
	if logger.level <= zapcore.DebugLevel {
		msg := fmt.Sprint(args...)
		logger.write(zapcore.DebugLevel, msg, nil)
	}
}

// Debugf uses the fmt.Sprintf to log a templated message.
func (logger *Logger) Debugf(template string, args ...interface{}) {
	if logger.level <= zapcore.DebugLevel {
		msg := fmt.Sprintf(template, args...)
		logger.write(zapcore.DebugLevel, msg, nil)
	}
}

// Debugw logs a message with some additional context.
func (logger *Logger) Debugw(message string, fields ...zapcore.Field) {
	if logger.level <= zapcore.DebugLevel {
		logger.write(zapcore.DebugLevel, message, fields)
	}
}

// Info uses the fmt.Sprint to construct and log a message.
func (logger *Logger) Info(args ...interface{}) {
	if logger.level <= zapcore.InfoLevel {
		msg := fmt.Sprint(args...)
		logger.write(zapcore.InfoLevel, msg, nil)
	}
}

// Infof uses the fmt.Sprintf to log a templated message.
func (logger *Logger) Infof(template string, args ...interface{}) {
	if logger.level <= zapcore.InfoLevel {
		msg := fmt.Sprintf(template, args...)
		logger.write(zapcore.InfoLevel, msg, nil)
	}
}

// Infow logs a message with some additional context.
func (logger *Logger) Infow(message string, fields ...zapcore.Field) {
	if logger.level <= zapcore.InfoLevel {
		logger.write(zapcore.InfoLevel, message, fields)
	}
}

// Warn uses the fmt.Sprint to construct and log a message.
func (logger *Logger) Warn(args ...interface{}) {
	if logger.level <= zapcore.WarnLevel {
		msg := fmt.Sprint(args...)
		logger.write(zapcore.WarnLevel, msg, nil)
	}
}

// Warnf uses the fmt.Sprintf to log a templated message.
func (logger *Logger) Warnf(template string, args ...interface{}) {
	if logger.level <= zapcore.WarnLevel {
		msg := fmt.Sprintf(template, args...)
		logger.write(zapcore.WarnLevel, msg, nil)
	}
}

// Warnw logs a message with some additional context.
func (logger *Logger) Warnw(message string, fields ...zapcore.Field) {
	if logger.level <= zapcore.WarnLevel {
		logger.write(zapcore.WarnLevel, message, fields)
	}
}

// Error uses the fmt.Sprint to construct and log a message.
func (logger *Logger) Error(args ...interface{}) {
	if logger.level <= zapcore.ErrorLevel {
		msg := fmt.Sprint(args...)
		logger.write(zapcore.ErrorLevel, msg, nil)
	}
}

// Errorf uses the fmt.Sprintf to log a templated message.
func (logger *Logger) Errorf(template string, args ...interface{}) {
	if logger.level <= zapcore.ErrorLevel {
		msg := fmt.Sprintf(template, args...)
		logger.write(zapcore.ErrorLevel, msg, nil)
	}
}

// Errorw logs a message with some additional context.
func (logger *Logger) Errorw(message string, fields ...zapcore.Field) {
	if logger.level <= zapcore.ErrorLevel {
		logger.write(zapcore.ErrorLevel, message, fields)
	}
}

// Panic uses the fmt.Sprint to construct and log a message.
func (logger *Logger) Panic(args ...interface{}) {
	if logger.level <= zapcore.PanicLevel {
		msg := fmt.Sprint(args...)
		logger.write(zapcore.PanicLevel, msg, nil)
	}
}

// Panicf uses the fmt.Sprintf to log a templated message.
func (logger *Logger) Panicf(template string, args ...interface{}) {
	if logger.level <= zapcore.PanicLevel {
		msg := fmt.Sprintf(template, args...)
		logger.write(zapcore.PanicLevel, msg, nil)
	}
}

// Panicw logs a message with some additional context.
func (logger *Logger) Panicw(message string, fields ...zapcore.Field) {
	if logger.level <= zapcore.PanicLevel {
		logger.write(zapcore.PanicLevel, message, fields)
	}
}

// Fatal uses the fmt.Sprint to construct and log a message.
func (logger *Logger) Fatal(args ...interface{}) {
	if logger.level <= zapcore.FatalLevel {
		msg := fmt.Sprint(args...)
		logger.write(zapcore.FatalLevel, msg, nil)
	}
}

// Fatalf uses the fmt.Sprintf to log a templated message.
func (logger *Logger) Fatalf(template string, args ...interface{}) {
	if logger.level <= zapcore.FatalLevel {
		msg := fmt.Sprintf(template, args...)
		logger.write(zapcore.FatalLevel, msg, nil)
	}
}

// Fatalw logs a message with some additional context.
func (logger *Logger) Fatalw(message string, fields ...zapcore.Field) {
	if logger.level <= zapcore.FatalLevel {
		logger.write(zapcore.FatalLevel, message, fields)
	}
}

// NewLogger sets up a Logger object according to a series of options.
func NewLogger(opts ...Option) (*Logger, error) {
	var (
		writer zapcore.WriteSyncer
		enc    zapcore.Encoder
	)

	o := &options{
		logLevel:   "warn",
		outputFile: "stderr",
	}
	for _, opt := range opts {
		opt.apply(o)
	}

	level, ok := levelMap[o.logLevel]
	if !ok {
		return nil, fmt.Errorf("unknown log level %s", o.logLevel)
	}

	logger := &Logger{
		level: level,
	}

	if o.writeSyncer != nil {
		writer = o.writeSyncer
	} else {
		if o.outputFile == "stdout" {
			writer = os.Stdout
		} else if o.outputFile == "stderr" {
			writer = os.Stderr
		} else {
			file, err := os.OpenFile(o.outputFile, os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return nil, err
			}
			writer = file
		}
	}

	if writer == os.Stdout || writer == os.Stderr {
		enc = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			TimeKey:        "time",
			NameKey:        "context",
			CallerKey:      "caller",
			StacktraceKey:  "backtrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	} else {
		enc = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			TimeKey:        "time",
			NameKey:        "context",
			CallerKey:      "caller",
			StacktraceKey:  "backtrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	}
	logger.writer = writer
	logger.core = zapcore.NewCore(enc, writer, level)
	return logger, nil
}
