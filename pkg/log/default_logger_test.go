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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logHandler = map[string][]reflect.Value{
		zapcore.DebugLevel.String(): {
			reflect.ValueOf(Debug),
			reflect.ValueOf(Debugf),
			reflect.ValueOf(Debugw),
		},
		zapcore.InfoLevel.String(): {
			reflect.ValueOf(Info),
			reflect.ValueOf(Infof),
			reflect.ValueOf(Infow),
		},
		zapcore.WarnLevel.String(): {
			reflect.ValueOf(Warn),
			reflect.ValueOf(Warnf),
			reflect.ValueOf(Warnw),
		},
		zapcore.ErrorLevel.String(): {
			reflect.ValueOf(Error),
			reflect.ValueOf(Errorf),
			reflect.ValueOf(Errorw),
		},
		zapcore.PanicLevel.String(): {
			reflect.ValueOf(Panic),
			reflect.ValueOf(Panicf),
			reflect.ValueOf(Panicw),
		},
		zapcore.FatalLevel.String(): {
			reflect.ValueOf(Fatal),
			reflect.ValueOf(Fatalf),
			reflect.ValueOf(Fatalw),
		},
	}
)

func TestDefaultLogger(t *testing.T) {
	for level, handlers := range logHandler {
		t.Run("test log with level "+level, func(t *testing.T) {
			fws := &fakeWriteSyncer{}
			logger, err := NewLogger(WithLogLevel(level), WithWriteSyncer(fws))
			assert.Nil(t, err, "failed to new logger: ", err)
			defer logger.Close()
			// Reset default logger
			DefaultLogger = logger

			handlers[0].Call([]reflect.Value{reflect.ValueOf("hello")})
			assert.Nil(t, logger.Sync(), "failed to sync logger")

			fields := unmarshalLogMessage(t, fws.bytes())
			assert.Equal(t, fields.Level, level, "bad log level ", fields.Level)
			assert.Equal(t, fields.Message, "hello", "bad log message ", fields.Message)

			handlers[1].Call([]reflect.Value{reflect.ValueOf("hello I am %s"), reflect.ValueOf("alex")})
			assert.Nil(t, logger.Sync(), "failed to sync logger")

			fields = unmarshalLogMessage(t, fws.bytes())
			assert.Equal(t, fields.Level, level, "bad log level ", fields.Level)
			assert.Equal(t, fields.Message, "hello I am alex", "bad log message ", fields.Message)

			handlers[2].Call([]reflect.Value{reflect.ValueOf("hello"), reflect.ValueOf(zap.String("name", "alex")), reflect.ValueOf(zap.Int("age", 3))})

			assert.Nil(t, logger.Sync(), "failed to sync logger")

			fields = unmarshalLogMessage(t, fws.bytes())
			assert.Equal(t, fields.Level, level, "bad log level ", fields.Level)
			assert.Equal(t, fields.Message, "hello", "bad log message ", fields.Message)
			assert.Equal(t, fields.Name, "alex", "bad name field ", fields.Name)
			assert.Equal(t, fields.Age, 3, "bad age field ", fields.Age)
		})
	}
}
