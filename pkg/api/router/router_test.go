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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
)

func TestHealthz(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	mountHealthz(r)
	healthz(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp healthzResponse
	dec := json.NewDecoder(w.Body)
	assert.Nil(t, dec.Decode(&resp))

	assert.Equal(t, healthzResponse{Status: "ok"}, resp)
}

func TestApisixHealthz(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	var state HealthState
	MountApisixHealthz(r, &state)
	apisixHealthz(&state)(c)

	assert.Equal(t, w.Code, http.StatusOK)

	var resp healthzResponse
	dec := json.NewDecoder(w.Body)
	assert.Nil(t, dec.Decode(&resp))

	assert.Equal(t, resp, healthzResponse{Status: "ok"})
}

func TestMetrics(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	req, err := http.NewRequest("GET", "/metrics", nil)
	assert.Nil(t, err, nil)
	c.Request = req
	mountMetrics(r)
	metrics(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebhooks(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	req, err := http.NewRequest("POST", "/validation", nil)
	assert.Nil(t, err, nil)
	c.Request = req
	MountWebhooks(r, &apisix.ClusterOptions{})

	assert.Equal(t, http.StatusOK, w.Code)
}
