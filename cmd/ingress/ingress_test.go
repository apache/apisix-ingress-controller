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
	"bytes"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/api7/ingress-controller/conf"
	"github.com/api7/ingress-controller/pkg/log"
)

type fakeWriteSyncer struct {
	buf bytes.Buffer
}

type fields struct {
	Level   string
	Time    string
	Message string
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

func TestSignalHandler(t *testing.T) {
	// TODO remove these two lines.
	conf.ENV = "local"
	conf.SetConfPath("./testdata/conf.json")
	conf.Init()
	cmd := NewIngressCommand()
	waitCh := make(chan struct{})
	go func() {
		cmd.Run(cmd, nil)
		close(waitCh)
	}()

	time.Sleep(time.Second)
	fws := &fakeWriteSyncer{}
	logger, err := log.NewLogger(log.WithLogLevel("info"), log.WithWriteSyncer(fws))
	assert.Nil(t, err)
	defer logger.Close()
	log.DefaultLogger = logger

	assert.Nil(t, syscall.Kill(os.Getpid(), syscall.SIGINT))
	<-waitCh

	msg := string(fws.buf.Bytes())
	assert.Contains(t, msg, fmt.Sprintf("signal %d (%s) received", syscall.SIGINT, syscall.SIGINT.String()))
	assert.Contains(t, msg, "apisix-ingress-controller exited")
}
