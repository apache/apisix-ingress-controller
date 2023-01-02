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
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GinLogger receive gin The default log of the framework
func GinLogger(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()
		cost := time.Since(start)
		if c.Writer.Status() >= 400 && c.Writer.Status() <= 599 {
			logger.Errorf("path: %s, status: %d, method: %s, query: %s, ip: %s, user-agent: %s, errors: %s, cost: %s",
				path, c.Writer.Status(), c.Request.Method, query, c.ClientIP(), c.Request.UserAgent(), c.Errors.ByType(gin.ErrorTypePrivate).String(), cost)
		} else {
			logger.Debugf("path: %s, status: %d, method: %s, query: %s, ip: %s, user-agent: %s, errors: %s, cost: %s",
				path, c.Writer.Status(), c.Request.Method, query, c.ClientIP(), c.Request.UserAgent(), c.Errors.ByType(gin.ErrorTypePrivate).String(), cost)
		}
	}
}

// GinRecovery recover Drop the project that may appear panic
func GinRecovery(logger *Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					logger.Errorf("path: %s, error: %s, request: %s",
						c.Request.URL.Path, err, string(httpRequest))
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()

					return
				}
				if stack {
					logger.Errorf("error: [Recovery from panic] %s, request: %s, stack: %s",
						err, string(httpRequest), string(debug.Stack()),
					)
				} else {
					logger.Errorf("error: [Recovery from panic] %s, request: %s",
						err, string(httpRequest))
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
