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
package api

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/config"
)

func TestServer(t *testing.T) {
	cfg := &config.Config{HTTPListen: "127.0.0.1:0"}
	srv, err := NewServer(cfg)
	assert.Nil(t, err, "see non-nil error: ", err)

	err = srv.httpListener.Close()
	assert.Nil(t, err, "see non-nil error: ", err)
}

func TestServerRun(t *testing.T) {
	cfg := &config.Config{HTTPListen: "127.0.0.1:0"}
	srv, err := NewServer(cfg)
	assert.Nil(t, err, "see non-nil error: ", err)

	stopCh := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)
		close(stopCh)
	}()

	err = srv.Run(stopCh)
	assert.Nil(t, err, "see non-nil error: ", err)
}

func TestProfileNotMount(t *testing.T) {
	cfg := &config.Config{HTTPListen: "127.0.0.1:0"}
	srv, err := NewServer(cfg)
	assert.Nil(t, err, "see non-nil error: ", err)
	stopCh := make(chan struct{})
	go func() {
		err := srv.Run(stopCh)
		assert.Nil(t, err, "see non-nil error: ", err)
	}()

	u := (&url.URL{
		Scheme: "http",
		Host:   srv.httpListener.Addr().String(),
		Path:   "/debug/pprof/cmdline",
	}).String()

	resp, err := http.Get(u)
	assert.Nil(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusNotFound)
	close(stopCh)
}

func TestProfile(t *testing.T) {
	cfg := &config.Config{HTTPListen: "127.0.0.1:0", EnableProfiling: true}
	srv, err := NewServer(cfg)
	assert.Nil(t, err, "see non-nil error: ", err)
	stopCh := make(chan struct{})
	go func() {
		err := srv.Run(stopCh)
		assert.Nil(t, err, "see non-nil error: ", err)
	}()

	u := (&url.URL{
		Scheme: "http",
		Host:   srv.httpListener.Addr().String(),
		Path:   "/debug/pprof/cmdline",
	}).String()

	resp, err := http.Get(u)
	assert.Nil(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	close(stopCh)
}
