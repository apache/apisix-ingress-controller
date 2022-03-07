package log

import (
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

func GinLogger(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()
		cost := time.Since(start)

		logger.Infof("path: %s, status: %d, method: %s, query: %s, ip: %s, user-agent: %s, errors: %s, cost: %s",
			path, c.Writer.Status(), c.Request.Method, query, c.ClientIP(), c.Request.UserAgent(), c.Errors.ByType(gin.ErrorTypePrivate).String(), cost)
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

