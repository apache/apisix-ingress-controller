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
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type healthzResponse struct {
	Status string `json:"status"`
}

func mountHealthz(r *gin.Engine) {
	r.GET("/healthz", healthz)
	r.GET("/apisix/healthz", healthz)
}

func healthz(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusOK, healthzResponse{Status: "ok"})
	return
}

func mountMetrics(r *gin.Engine) {
	r.GET("/metrics", metrics)
}

func metrics(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}

// Mount mounts all api routers.
func Mount(r *gin.Engine) {
	mountHealthz(r)
	mountMetrics(r)
}
