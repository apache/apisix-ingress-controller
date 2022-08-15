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
package ingress

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type fakeWriteSyncer struct {
	buf bytes.Buffer
}

type fields struct {
	Level   string
	Time    string
	Message string
}

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

func (fws *fakeWriteSyncer) Sync() error {
	return nil
}

func (fws *fakeWriteSyncer) Write(p []byte) (int, error) {
	return fws.buf.Write(p)
}

func getRandomListen() string {
	port := rand.Intn(10000) + 10000
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func TestSignalHandler(t *testing.T) {
	cmd := NewIngressCommand()
	listen := getRandomListen()
	cmd.SetArgs([]string{
		"--log-level", "debug",
		"--log-output", "stderr",
		"--http-listen", listen,
		"--enable-profiling",
		"--kubeconfig", "/foo/bar/baz",
		"--resync-interval", "24h",
		"--default-apisix-cluster-base-url", "http://apisixgw.default.cluster.local/apisix",
		"--default-apisix-cluster-admin-key", "0x123",
	})
	waitCh := make(chan struct{})
	go func() {
		if err := cmd.Execute(); err != nil {
			log.Errorf("failed to execute command: %s", err)
		}
		close(waitCh)
	}()

	time.Sleep(5 * time.Second)
	fws := &fakeWriteSyncer{}
	logger, err := log.NewLogger(log.WithLogLevel("info"), log.WithWriteSyncer(fws))
	assert.Nil(t, err)
	defer logger.Close()
	log.DefaultLogger = logger

	assert.Nil(t, syscall.Kill(os.Getpid(), syscall.SIGINT))
	<-waitCh

	msg := fws.buf.String()
	assert.Contains(t, msg, fmt.Sprintf("signal %d (%s) received", syscall.SIGINT, syscall.SIGINT.String()))
	assert.Contains(t, msg, "apisix ingress controller exited")
}

func TestNewIngressCommandEffectiveLog(t *testing.T) {
	listen := getRandomListen()
	cmd := NewIngressCommand()
	cmd.SetArgs([]string{
		"--log-level", "debug",
		"--log-output", "./test.log",
		"--http-listen", listen,
		"--enable-profiling",
		"--kubeconfig", "/foo/bar/baz",
		"--resync-interval", "24h",
		"--default-apisix-cluster-base-url", "http://apisixgw.default.cluster.local/apisix",
		"--default-apisix-cluster-admin-key", "0x123",
	})
	defer os.Remove("./test.log")

	stopCh := make(chan struct{})
	go func() {
		assert.Nil(t, cmd.Execute())
		close(stopCh)
	}()

	time.Sleep(5 * time.Second)
	assert.Nil(t, syscall.Kill(os.Getpid(), syscall.SIGINT))
	<-stopCh

	file, err := os.Open("./test.log")
	assert.Nil(t, err)

	buf := bufio.NewReader(file)
	f := parseLog(t, buf)
	assert.Contains(t, f.Message, "apisix ingress controller started")
	assert.Equal(t, "info", f.Level)

	f = parseLog(t, buf)
	assert.Contains(t, f.Message, "version:")
	assert.Equal(t, "info", f.Level)

	f = parseLog(t, buf)
	assert.Contains(t, f.Message, "use configuration")
	assert.Equal(t, "info", f.Level)

	var cfg config.Config
	data := strings.TrimPrefix(f.Message, "use configuration\n")
	err = json.Unmarshal([]byte(data), &cfg)
	assert.Nil(t, err)

	assert.Equal(t, "./test.log", cfg.LogOutput)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, listen, cfg.HTTPListen)
	assert.Equal(t, true, cfg.EnableProfiling)
	assert.Equal(t, "/foo/bar/baz", cfg.Kubernetes.Kubeconfig)
	assert.Equal(t, types.TimeDuration{Duration: 24 * time.Hour}, cfg.Kubernetes.ResyncInterval)
	assert.Equal(t, "0x123", cfg.APISIX.DefaultClusterAdminKey)
	assert.Equal(t, "http://apisixgw.default.cluster.local/apisix", cfg.APISIX.DefaultClusterBaseURL)
}

func parseLog(t *testing.T, r *bufio.Reader) *fields {
	line, isPrefix, err := r.ReadLine()
	assert.False(t, isPrefix)
	assert.Nil(t, err)

	var f fields
	err = json.Unmarshal(line, &f)
	assert.Nil(t, err)
	return &f
}

func TestRotateLog(t *testing.T) {
	listen := getRandomListen()
	cmd := NewIngressCommand()
	cmd.SetArgs([]string{
		"--log-rotate-output-path", "./testlog/test.log",
		"--log-rotate-max-size", "1",
		"--log-level", "debug",
		"--log-output", "./testlog/test.log",
		"--http-listen", listen,
		"--enable-profiling",
		"--kubeconfig", "/foo/bar/baz",
		"--resync-interval", "24h",
		"--default-apisix-cluster-base-url", "http://apisixgw.default.cluster.local/apisix",
		"--default-apisix-cluster-admin-key", "0x123",
	})
	defer os.RemoveAll("./testlog/")

	stopCh := make(chan struct{})
	go func() {
		assert.Nil(t, cmd.Execute())
		close(stopCh)
	}()

	fws := &fakeWriteSyncer{}
	logger, err := log.NewLogger(log.WithLogLevel("debug"), log.WithWriteSyncer(fws))
	assert.Nil(t, err)
	defer logger.Close()
	log.DefaultLogger = logger

	// fill logs with data until the size > 1m
	line := ""
	for i := 0; i < 256; i++ {
		line += "0"
	}

	for i := 0; i < 4096; i++ {
		log.Debug(line)
	}

	time.Sleep(5 * time.Second)
	assert.Nil(t, syscall.Kill(os.Getpid(), syscall.SIGINT))
	<-stopCh

	files, err := ioutil.ReadDir("./testlog")

	if err != nil {
		t.Fatalf("Unable to read log dir: %v", err)
	}

	assert.Equal(t, true, len(files) >= 2)
}
