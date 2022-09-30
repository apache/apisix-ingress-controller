// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package log

import "go.uber.org/zap/zapcore"

var (
	// DefaultLogger is the default logger, which logs message to stderr and with
	// the minimal level "warn".
	DefaultLogger *Logger
)

func init() {
	l, err := NewLogger(
		WithOutputFile("stderr"),
		WithLogLevel("warn"),
	)
	if err != nil {
		panic(err)
	}
	DefaultLogger = l
}

// Level returns the DefaultLogger log level
func Level() zapcore.Level {
	return DefaultLogger.Level()
}

// Debug uses the fmt.Sprint to construct and log a message using the DefaultLogger.
func Debug(args ...interface{}) {
	DefaultLogger.Debug(args...)
}

// Debugf uses the fmt.Sprintf to log a templated message using the DefaultLogger.
func Debugf(template string, args ...interface{}) {
	DefaultLogger.Debugf(template, args...)
}

// Debugw logs a message with some additional context using the DefaultLogger.
func Debugw(message string, fields ...zapcore.Field) {
	DefaultLogger.Debugw(message, fields...)
}

// Info uses the fmt.Sprint to construct and log a message using the DefaultLogger.
func Info(args ...interface{}) {
	DefaultLogger.Info(args...)
}

// Infof uses the fmt.Sprintf to log a templated message using the DefaultLogger.
func Infof(template string, args ...interface{}) {
	DefaultLogger.Infof(template, args...)
}

// Infow logs a message with some additional context using the DefaultLogger.
func Infow(message string, fields ...zapcore.Field) {
	DefaultLogger.Infow(message, fields...)
}

// Warn uses the fmt.Sprint to construct and log a message using the DefaultLogger.
func Warn(args ...interface{}) {
	DefaultLogger.Warn(args...)
}

// Warnf uses the fmt.Sprintf to log a templated message using the DefaultLogger.
func Warnf(template string, args ...interface{}) {
	DefaultLogger.Warnf(template, args...)
}

// Warnw logs a message with some additional context using the DefaultLogger.
func Warnw(message string, fields ...zapcore.Field) {
	DefaultLogger.Warnw(message, fields...)
}

// Error uses the fmt.Sprint to construct and log a message using the DefaultLogger.
func Error(args ...interface{}) {
	DefaultLogger.Error(args...)
}

// Errorf uses the fmt.Sprintf to log a templated message using the DefaultLogger.
func Errorf(template string, args ...interface{}) {
	DefaultLogger.Errorf(template, args...)
}

// Errorw logs a message with some additional context using the DefaultLogger.
func Errorw(message string, fields ...zapcore.Field) {
	DefaultLogger.Errorw(message, fields...)
}

// Panic uses the fmt.Sprint to construct and log a message using the DefaultLogger.
func Panic(args ...interface{}) {
	DefaultLogger.Panic(args...)
}

// Panicf uses the fmt.Sprintf to log a templated message using the DefaultLogger.
func Panicf(template string, args ...interface{}) {
	DefaultLogger.Panicf(template, args...)
}

// Panicw logs a message with some additional context using the DefaultLogger.
func Panicw(message string, fields ...zapcore.Field) {
	DefaultLogger.Panicw(message, fields...)
}

// Fatal uses the fmt.Sprint to construct and log a message using the DefaultLogger.
func Fatal(args ...interface{}) {
	DefaultLogger.Fatal(args...)
}

// Fatalf uses the fmt.Sprintf to log a templated message using the DefaultLogger.
func Fatalf(template string, args ...interface{}) {
	DefaultLogger.Fatalf(template, args...)
}

// Fatalw logs a message with some additional context using the DefaultLogger.
func Fatalw(message string, fields ...zapcore.Field) {
	DefaultLogger.Fatalw(message, fields...)
}
