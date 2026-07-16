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

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/streaming/pkg/httpstream"
)

// Tunnel is the subset of a port-forward tunnel used by the scaffold.
type Tunnel interface {
	Endpoint() string
	Close()
}

// wsTunnel is a service port-forward that negotiates the connection over
// WebSockets first, falling back to SPDY. Kubernetes >= 1.31 apiservers are
// deprecating SPDY, and terratest's SPDY-only tunnel tears down completely on a
// single data-connection reset (common when APISIX's stream proxy closes a
// connection), which breaks the raw-TCP/L4 e2e tests. The WebSocket tunneling
// path is far more resilient, matching `kubectl port-forward`.
type wsTunnel struct {
	localPort int
	stopChan  chan struct{}
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

// newWebsocketServiceTunnel port-forwards servicePort of the named Service using a
// WebSocket-over-SPDY fallback dialer, and blocks until the tunnel is ready.
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

	// Resolve the Service to a ready pod and the pod-side target port.
	service, err := k8s.GetServiceE(t, opts, serviceName)
	if err != nil {
		return nil, err
	}
	pods, err := k8s.ListPodsE(t, opts, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(service.Spec.Selector).String(),
	})
	if err != nil {
		return nil, err
	}
	var podName string
	for i := range pods {
		if k8s.IsPodAvailable(&pods[i]) {
			podName = pods[i].Name
			break
		}
	}
	if podName == "" {
		return nil, fmt.Errorf("no available pod for service %s", serviceName)
	}
	targetPort := servicePort
	for _, p := range service.Spec.Ports {
		if int(p.Port) == servicePort {
			if p.TargetPort.Type == intstr.Int {
				targetPort = p.TargetPort.IntValue()
			}
			break
		}
	}

	localPort, err := k8s.GetAvailablePortE(t)
	if err != nil {
		return nil, err
	}

	pfURL := clientset.CoreV1().RESTClient().Post().
		Resource("pods").Namespace(opts.Namespace).Name(podName).
		SubResource("portforward").URL()

	spdyRT, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	spdyDialer := spdy.NewDialer(upgrader, &http.Client{Transport: spdyRT}, "POST", pfURL)
	wsDialer, err := portforward.NewSPDYOverWebsocketDialer(pfURL, config)
	if err != nil {
		return nil, err
	}
	dialer := portforward.NewFallbackDialer(wsDialer, spdyDialer, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})

	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, targetPort)}, stopChan, readyChan, io.Discard, io.Discard)
	if err != nil {
		return nil, err
	}
	errChan := make(chan error, 1)
	go func() { errChan <- fw.ForwardPorts() }()
	select {
	case <-readyChan:
		return &wsTunnel{localPort: localPort, stopChan: stopChan}, nil
	case err := <-errChan:
		return nil, fmt.Errorf("failed to establish port forward to service %s: %w", serviceName, err)
	}
}
