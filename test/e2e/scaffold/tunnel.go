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
	"io"
	"net/http"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/streaming/pkg/httpstream"
)

// Tunnel is the subset of a port-forward tunnel used by the scaffold.
type Tunnel interface {
	Endpoint() string
	Close()
}

// wsTunnel is a self-healing service port-forward that negotiates the connection
// over WebSockets first, falling back to SPDY. Two properties make it fit the
// raw-TCP/L4 e2e tests where terratest's SPDY-only tunnel is unreliable against
// Kubernetes >= 1.31:
//   - WebSocket tunneling (as `kubectl port-forward` uses) survives data-connection
//     resets that tear down a SPDY tunnel — APISIX's stream proxy triggers these by
//     closing connections.
//   - It re-establishes the forward (re-resolving the backing pod) whenever
//     ForwardPorts returns, so the local port keeps listening across transient
//     drops or an APISIX pod restart, instead of dying permanently.
type wsTunnel struct {
	t           testing.TestingT
	opts        *k8s.KubectlOptions
	clientset   *kubernetes.Clientset
	config      *rest.Config
	serviceName string
	servicePort int
	localPort   int
	stopChan    chan struct{}
}

// Endpoint returns the local IPv4 address the tunnel listens on.
func (t *wsTunnel) Endpoint() string { return fmt.Sprintf("127.0.0.1:%d", t.localPort) }

func (t *wsTunnel) Close() {
	select {
	case <-t.stopChan:
	default:
		close(t.stopChan)
	}
}

func (t *wsTunnel) stopped() bool {
	select {
	case <-t.stopChan:
		return true
	default:
		return false
	}
}

// forwardOnce establishes a single port-forward session and blocks until it ends
// (tunnel closed or a forwarding error). ready is closed once the local port is
// listening; it is only meaningful on the first successful call.
func (t *wsTunnel) forwardOnce(ready chan<- struct{}) error {
	service, err := k8s.GetServiceE(t.t, t.opts, t.serviceName)
	if err != nil {
		return err
	}
	pods, err := k8s.ListPodsE(t.t, t.opts, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(service.Spec.Selector).String(),
	})
	if err != nil {
		return err
	}
	var podName string
	for i := range pods {
		if k8s.IsPodAvailable(&pods[i]) {
			podName = pods[i].Name
			break
		}
	}
	if podName == "" {
		return fmt.Errorf("no available pod for service %s", t.serviceName)
	}
	targetPort := t.servicePort
	for _, p := range service.Spec.Ports {
		if int(p.Port) == t.servicePort {
			if p.TargetPort.Type == intstr.Int {
				targetPort = p.TargetPort.IntValue()
			}
			break
		}
	}

	pfURL := t.clientset.CoreV1().RESTClient().Post().
		Resource("pods").Namespace(t.opts.Namespace).Name(podName).
		SubResource("portforward").URL()

	spdyRT, upgrader, err := spdy.RoundTripperFor(t.config)
	if err != nil {
		return err
	}
	spdyDialer := spdy.NewDialer(upgrader, &http.Client{Transport: spdyRT}, "POST", pfURL)
	wsDialer, err := portforward.NewSPDYOverWebsocketDialer(pfURL, t.config)
	if err != nil {
		return err
	}
	dialer := portforward.NewFallbackDialer(wsDialer, spdyDialer, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})

	readyChan := make(chan struct{})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", t.localPort, targetPort)}, t.stopChan, readyChan, io.Discard, io.Discard)
	if err != nil {
		return err
	}
	errChan := make(chan error, 1)
	go func() { errChan <- fw.ForwardPorts() }()
	select {
	case <-readyChan:
		if ready != nil {
			close(ready)
		}
		return <-errChan
	case err := <-errChan:
		return err
	}
}

// newWebsocketServiceTunnel port-forwards servicePort of the named Service and
// keeps it alive until Close, re-establishing on failure. It blocks until the
// first session is ready.
func newWebsocketServiceTunnel(t testing.TestingT, opts *k8s.KubectlOptions, serviceName string, servicePort int) (*wsTunnel, error) {
	clientset, err := k8s.GetKubernetesClientFromOptionsE(t, opts)
	if err != nil {
		return nil, err
	}
	config := opts.RestConfig
	if config == nil {
		path, err := opts.GetConfigPath(t)
		if err != nil {
			return nil, err
		}
		if config, err = k8s.LoadApiClientConfigE(path, opts.ContextName); err != nil {
			return nil, err
		}
	}
	localPort, err := k8s.GetAvailablePortE(t)
	if err != nil {
		return nil, err
	}

	tunnel := &wsTunnel{
		t:           t,
		opts:        opts,
		clientset:   clientset,
		config:      config,
		serviceName: serviceName,
		servicePort: servicePort,
		localPort:   localPort,
		stopChan:    make(chan struct{}),
	}

	ready := make(chan struct{})
	firstErr := make(chan error, 1)
	go func() {
		first := true
		for !tunnel.stopped() {
			var r chan<- struct{}
			if first {
				r = ready
			}
			err := tunnel.forwardOnce(r)
			if first {
				select {
				case <-ready:
					// became ready at least once; from here on just keep healing
				default:
					firstErr <- err
					return
				}
				first = false
			}
			if tunnel.stopped() {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	select {
	case <-ready:
		return tunnel, nil
	case err := <-firstErr:
		return nil, fmt.Errorf("failed to establish port forward to service %s: %w", serviceName, err)
	}
}
