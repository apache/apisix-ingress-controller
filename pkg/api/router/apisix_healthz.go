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
)

// MountApisixHealthz mounts apisix healthz route.
func MountApisixHealthz(r *gin.Engine, state *HealthState) {
	r.GET("/apisix/healthz", apisixHealthz(state))
}

func apisixHealthz(state *HealthState) gin.HandlerFunc {
	return func(c *gin.Context) {
		state.RLock()
		err := state.Err
		state.RUnlock()

		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError,
				healthzResponse{Status: err.Error()})
			return
		}
		c.AbortWithStatusJSON(http.StatusOK, healthzResponse{Status: "ok"})
	}
}
