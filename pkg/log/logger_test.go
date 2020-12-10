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
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type fakeWriteSyncer struct {
	buf bytes.Buffer
}

type fields struct {
	Level   string
	Time    string
	Message string
	Name    string
	Age     int
}

func (fws *fakeWriteSyncer) Sync() error {
	return nil
}

func (fws *fakeWriteSyncer) Write(p []byte) (int, error) {
	return fws.buf.Write(p)
}

func (fws *fakeWriteSyncer) bytes() (p []byte) {
	s := fws.buf.Bytes()
	p = make([]byte, len(s))
	copy(p, s)
	fws.buf.Reset()
	return
}

func unmarshalLogMessage(t *testing.T, data []byte) *fields {
	var f fields
	err := json.Unmarshal(data, &f)
	assert.Nil(t, err, "failed to unmarshal log message: ", err)
	return &f
}

func TestLogger(t *testing.T) {
	for level := range levelMap {
		t.Run("test log with level "+level, func(t *testing.T) {
			fws := &fakeWriteSyncer{}
			logger, err := NewLogger(WithLogLevel(level), WithWriteSyncer(fws))
			assert.Nil(t, err, "failed to new logger: ", err)
			defer logger.Close()

			rv := reflect.ValueOf(logger)

			handler := rv.MethodByName(http.CanonicalHeaderKey(level))
			handler.Call([]reflect.Value{reflect.ValueOf("hello")})

			assert.Nil(t, logger.Sync(), "failed to sync logger")

			fields := unmarshalLogMessage(t, fws.bytes())
			assert.Equal(t, fields.Level, level, "bad log level ", fields.Level)
			assert.Equal(t, fields.Message, "hello", "bad log message ", fields.Message)

			handler = rv.MethodByName(http.CanonicalHeaderKey(level) + "f")
			handler.Call([]reflect.Value{reflect.ValueOf("hello I am %s"), reflect.ValueOf("alex")})

			assert.Nil(t, logger.Sync(), "failed to sync logger")

			fields = unmarshalLogMessage(t, fws.bytes())
			assert.Equal(t, fields.Level, level, "bad log level ", fields.Level)
			assert.Equal(t, fields.Message, "hello I am alex", "bad log message ", fields.Message)

			handler = rv.MethodByName(http.CanonicalHeaderKey(level) + "w")
			handler.Call([]reflect.Value{reflect.ValueOf("hello"), reflect.ValueOf(zap.String("name", "alex")), reflect.ValueOf(zap.Int("age", 3))})

			assert.Nil(t, logger.Sync(), "failed to sync logger")

			fields = unmarshalLogMessage(t, fws.bytes())
			assert.Equal(t, fields.Level, level, "bad log level ", fields.Level)
			assert.Equal(t, fields.Message, "hello", "bad log message ", fields.Message)
			assert.Equal(t, fields.Name, "alex", "bad name field ", fields.Name)
			assert.Equal(t, fields.Age, 3, "bad age field ", fields.Age)
		})
	}
}

func TestLogLevel(t *testing.T) {
	fws := &fakeWriteSyncer{}
	logger, err := NewLogger(WithLogLevel("error"), WithWriteSyncer(fws))
	assert.Nil(t, err, "failed to new logger: ", err)
	defer logger.Close()

	logger.Warn("this message should be dropped")
	assert.Nil(t, logger.Sync(), "failed to sync logger")

	p := fws.bytes()
	assert.Len(t, p, 0, "saw a message which should be dropped")
}
