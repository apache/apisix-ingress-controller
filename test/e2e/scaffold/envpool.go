// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package scaffold

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/gruntwork-io/terratest/modules/k8s"
	corev1 "k8s.io/api/core/v1"
)

// pooledEnv is a fully provisioned, ready-to-use test environment produced by
// the prewarm pool ahead of time, so a spec's BeforeEach can pick it up without
// paying the deploy/readiness latency on its critical path.
type pooledEnv struct {
	namespace        string
	kubectlOptions   *k8s.KubectlOptions
	controllerName   string
	adminKey         string
	dataplaneService *corev1.Service
	httpbinService   *corev1.Service

	// err is non-nil if provisioning failed; the consumer falls back to a
	// synchronous deploy and discards this env.
	err error
}

// bgTestingT is a minimal terratest testing.TestingT implementation usable from
// background goroutines (the prewarm workers run outside any Ginkgo spec, so
// Ginkgo's GinkgoT/Expect must not be used there). Fatal/FailNow abort the
// current provision via panic, which safeProvision recovers into pooledEnv.err.
type bgTestingT struct{}

type bgAbort struct{ msg string }

func (bgAbort) Error() string { return "prewarm provision aborted" }

func (t *bgTestingT) Fail()                        {}
func (t *bgTestingT) FailNow()                     { panic(bgAbort{msg: "FailNow"}) }
func (t *bgTestingT) Fatal(args ...any)            { panic(bgAbort{msg: fmt.Sprint(args...)}) }
func (t *bgTestingT) Fatalf(f string, args ...any) { panic(bgAbort{msg: fmt.Sprintf(f, args...)}) }
func (t *bgTestingT) Error(args ...any)            {}
func (t *bgTestingT) Errorf(f string, args ...any) {}
func (t *bgTestingT) Name() string                 { return "prewarm" }

// envPool maintains a small set of prewarmed environments for one profile.
// A buffered channel of capacity `depth` holds ready environments; `depth`
// worker goroutines keep refilling it. With depth=1 this is a double buffer:
// one env is ready while the next is being built concurrently with the
// currently running spec.
type envPool struct {
	ch        chan *pooledEnv
	stop      chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup
	provision func() *pooledEnv
}

func newEnvPool(depth int, provision func() *pooledEnv) *envPool {
	if depth < 1 {
		depth = 1
	}
	p := &envPool{
		ch:        make(chan *pooledEnv, depth),
		stop:      make(chan struct{}),
		provision: provision,
	}
	for i := 0; i < depth; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

func (p *envPool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.stop:
			return
		default:
		}
		env := safeProvision(p.provision)
		select {
		case p.ch <- env:
		case <-p.stop:
			// Shutting down before this env was consumed: clean it up.
			destroyPooledEnv(env)
			return
		}
	}
}

func (p *envPool) acquire() *pooledEnv {
	select {
	case env := <-p.ch:
		return env
	case <-p.stop:
		return nil
	}
}

func (p *envPool) shutdown() {
	p.stopOnce.Do(func() { close(p.stop) })
	p.wg.Wait()
	// Drain and destroy any environments still buffered.
	for {
		select {
		case env := <-p.ch:
			destroyPooledEnv(env)
		default:
			return
		}
	}
}

// safeProvision runs a provision function, converting any panic (including a
// bgTestingT Fatal/FailNow) into a pooledEnv carrying the error.
func safeProvision(provision func() *pooledEnv) (env *pooledEnv) {
	defer func() {
		if r := recover(); r != nil {
			if env == nil {
				env = &pooledEnv{}
			}
			env.err = fmt.Errorf("panic during prewarm provision: %v", r)
		}
	}()
	return provision()
}

// destroyPooledEnv tears down an environment that will not be used by a spec by
// deleting its namespace. Tunnels are only created once an env is handed to a
// spec (see loadPooledEnv), so an unused pooled env owns no port-forwards.
func destroyPooledEnv(env *pooledEnv) {
	if env == nil {
		return
	}
	if env.namespace != "" && env.kubectlOptions != nil {
		_ = k8s.RunKubectlE(&bgTestingT{}, env.kubectlOptions, "delete", "namespace", env.namespace, "--wait=false")
	}
}

// --- process-global pool registry, keyed by profile -------------------------

var (
	poolsMu sync.Mutex
	pools   = map[string]*envPool{}
)

func getOrStartPool(key string, depth int, provision func() *pooledEnv) *envPool {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	if p, ok := pools[key]; ok {
		return p
	}
	p := newEnvPool(depth, provision)
	pools[key] = p
	return p
}

// ShutdownAllPools stops every prewarm pool and tears down any environments
// still buffered. It is registered as an AfterSuite by the e2e suite root, not
// here, so that suites which merely import this package (e.g. the benchmark
// suite, which declares its own AfterSuite) are not given a duplicate node.
func ShutdownAllPools() {
	poolsMu.Lock()
	ps := pools
	pools = map[string]*envPool{}
	poolsMu.Unlock()
	for _, p := range ps {
		p.shutdown()
	}
}

// --- knobs ------------------------------------------------------------------

// prewarmEnabled reports whether the prewarm pool is active. It is on by
// default and can be disabled with E2E_PREWARM=false.
func prewarmEnabled() bool {
	return getEnvOrDefault("E2E_PREWARM", "true") != "false"
}

// prewarmDepth is the pool depth (ready + in-flight environments) per profile
// per process. Default 1 (double buffer). Override with E2E_PREWARM_DEPTH.
func prewarmDepth() int {
	if v := os.Getenv("E2E_PREWARM_DEPTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 1
}
