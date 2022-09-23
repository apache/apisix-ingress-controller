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

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const TestMode = "test"

func init() {
	gin.SetMode(TestMode)
}

type header struct {
	Key   string
	Value string
}

func performRequest(r http.Handler, method, path string, headers ...header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for _, h := range headers {
		req.Header.Add(h.Key, h.Value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGinLogger(t *testing.T) {
	t.Run("test log with gin logger", func(t *testing.T) {
		fws := &fakeWriteSyncer{}
		logger, err := NewLogger(WithLogLevel("debug"), WithWriteSyncer(fws))
		assert.Nil(t, err, "failed to new logger: ", err)
		defer logger.Close()

		router := gin.New()
		router.Use(GinLogger(logger))
		router.GET("/healthz", func(c *gin.Context) {})
		performRequest(router, "GET", "/healthz")
		res := string(fws.bytes())
		assert.Contains(t, res, "200")
		assert.Contains(t, res, "GET")
		assert.Contains(t, res, "/healthz")
	})
	t.Run("test log with gin logger 4xx and 5xx", func(t *testing.T) {
		fws := &fakeWriteSyncer{}
		logger, err := NewLogger(WithLogLevel("debug"), WithWriteSyncer(fws))
		assert.Nil(t, err, "failed to new logger: ", err)
		defer logger.Close()

		router := gin.New()
		router.Use(GinLogger(logger))
		router.GET("/healthz", func(c *gin.Context) { c.JSON(500, nil) })
		performRequest(router, "GET", "/healthz")
		res := string(fws.bytes())
		assert.Contains(t, res, "500")
		assert.Contains(t, res, "GET")
		assert.Contains(t, res, "/healthz")
		assert.Contains(t, res, "error")
		router.GET("/healthz-check", func(c *gin.Context) { c.JSON(400, nil) })
		performRequest(router, "GET", "/healthz-check")
		res = string(fws.bytes())
		assert.Contains(t, res, "400")
		assert.Contains(t, res, "GET")
		assert.Contains(t, res, "/healthz-check")
		assert.Contains(t, res, "error")
	})
}

func TestGinRecovery(t *testing.T) {
	t.Run("test log with gin recovery with stack", func(t *testing.T) {
		fws := &fakeWriteSyncer{}
		logger, err := NewLogger(WithLogLevel("debug"), WithWriteSyncer(fws))
		assert.Nil(t, err, "failed to new logger: ", err)
		defer logger.Close()

		router := gin.New()
		router.Use(GinRecovery(logger, true))
		router.GET("/healthz", func(c *gin.Context) { panic("test log with gin recovery") })
		performRequest(router, "GET", "/healthz")
		res := string(fws.bytes())
		fmt.Println(res)
		assert.Contains(t, res, "caller")
		assert.Contains(t, res, "GET")
		assert.Contains(t, res, "/healthz")
		assert.Contains(t, res, "stack")
	})
	t.Run("test log with gin recovery", func(t *testing.T) {
		fws := &fakeWriteSyncer{}
		logger, err := NewLogger(WithLogLevel("debug"), WithWriteSyncer(fws))
		assert.Nil(t, err, "failed to new logger: ", err)
		defer logger.Close()

		router := gin.New()
		router.Use(GinRecovery(logger, false))
		router.GET("/healthz", func(c *gin.Context) { panic("test log with gin recovery") })
		performRequest(router, "GET", "/healthz")
		res := string(fws.bytes())
		fmt.Println(res)
		assert.Contains(t, res, "caller")
		assert.Contains(t, res, "GET")
		assert.Contains(t, res, "/healthz")
		assert.NotContains(t, res, "stack")
	})
}
