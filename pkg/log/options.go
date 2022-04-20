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
