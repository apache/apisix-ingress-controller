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
package translation

import (
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	config "github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/pod"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	DefaultWeight = 100
)

type TranslateError struct {
	Field  string
	Reason string
}

func (te *TranslateError) Error() string {
	return fmt.Sprintf("%s: %s", te.Field, te.Reason)
}

type Translator interface {
	// TranslateUpstream composes an upstream according to the
	// given namespace, name (searching Service/Endpoints) and port (filtering Endpoints).
	// The returned Upstream doesn't have metadata info.
	// It doesn't assign any metadata fields, so it's caller's responsibility to decide
	// the metadata.
	// Note the subset is used to filter the ultimate node list, only pods whose labels
	// matching the subset labels (defined in ApisixUpstream) will be selected.
	// When the subset is not found, the node list will be empty. When the subset is empty,
	// all pods IP will be filled.
	TranslateService(string, string, string, int32) (*apisixv1.Upstream, error)
	// TranslateUpstreamNodes translate Endpoints resources to APISIX Upstream nodes
	// according to the give port. Extra labels can be passed to filter the ultimate
	// upstream nodes.
	TranslateEndpoint(kube.Endpoint, int32, types.Labels) (apisixv1.UpstreamNodes, error)

	TranslateAnnotations(anno map[string]string) apisixv1.Plugins

	// ExtractKeyPair extracts certificate and private key pair from secret
	// Supports APISIX style ("cert" and "key") and Kube style ("tls.crt" and "tls.key)
	ExtractKeyPair(s *corev1.Secret, hasPrivateKey bool) ([]byte, []byte, error)
}

// TranslatorOptions contains options to help Translator
// work well.
type TranslatorOptions struct {
	EndpointLister       kube.EndpointLister
	ServiceLister        listerscorev1.ServiceLister
	ApisixUpstreamLister kube.ApisixUpstreamLister
	APIVersion           string

	PodProvider pod.Provider
	PodLister   listerscorev1.PodLister
}

type translator struct {
	*TranslatorOptions
}

// NewTranslator initializes a APISIX CRD resources Translator.
func NewTranslator(opts *TranslatorOptions) Translator {
	return &translator{
		TranslatorOptions: opts,
	}
}

func (t *translator) TranslateService(namespace, name, subset string, port int32) (*apisixv1.Upstream, error) {
	endpoint, err := t.EndpointLister.GetEndpoint(namespace, name)
	if err != nil {
		return nil, &TranslateError{
			Field:  "endpoints",
			Reason: err.Error(),
		}
	}

	switch t.APIVersion {
	case config.ApisixV2beta3:
		return t.translateUpstreamV2beta3(&endpoint, namespace, name, subset, port)
	case config.ApisixV2:
		return t.translateUpstreamV2(&endpoint, namespace, name, subset, port)
	default:
		panic(fmt.Errorf("unsupported ApisixUpstream version %v", t.APIVersion))
	}
}

func (t *translator) translateUpstreamV2(ep *kube.Endpoint, namespace, name, subset string, port int32) (*apisixv1.Upstream, error) {
	au, err := t.ApisixUpstreamLister.V2(namespace, name)
	ups := apisixv1.NewDefaultUpstream()
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// If subset in ApisixRoute is not empty but the ApisixUpstream resource not found,
			// just set an empty node list.
			if subset != "" {
				ups.Nodes = apisixv1.UpstreamNodes{}
				return ups, nil
			}
		} else {
			return nil, &TranslateError{
				Field:  "ApisixUpstream",
				Reason: err.Error(),
			}
		}
	}
	var labels types.Labels
	if subset != "" {
		for _, ss := range au.V2().Spec.Subsets {
			if ss.Name == subset {
				labels = ss.Labels
				break
			}
		}
	}
	// Filter nodes by subset.
	nodes, err := t.TranslateEndpoint(*ep, port, labels)
	if err != nil {
		return nil, err
	}
	if au == nil || au.V2().Spec == nil {
		ups.Nodes = nodes
		return ups, nil
	}

	upsCfg := &au.V2().Spec.ApisixUpstreamConfig
	for _, pls := range au.V2().Spec.PortLevelSettings {
		if pls.Port == port {
			upsCfg = &pls.ApisixUpstreamConfig
			break
		}
	}
	ups, err = t.TranslateUpstreamConfigV2(upsCfg)
	if err != nil {
		return nil, err
	}
	ups.Nodes = nodes
	return ups, nil
}

func (t *translator) translateUpstreamV2beta3(ep *kube.Endpoint, namespace, name, subset string, port int32) (*apisixv1.Upstream, error) {
	au, err := t.ApisixUpstreamLister.V2beta3(namespace, name)
	ups := apisixv1.NewDefaultUpstream()
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// If subset in ApisixRoute is not empty but the ApisixUpstream resource not found,
			// just set an empty node list.
			if subset != "" {
				ups.Nodes = apisixv1.UpstreamNodes{}
				return ups, nil
			}
		} else {
			return nil, &TranslateError{
				Field:  "ApisixUpstream",
				Reason: err.Error(),
			}
		}
	}
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// If subset in ApisixRoute is not empty but the ApisixUpstream resource not found,
			// just set an empty node list.
			if subset != "" {
				ups.Nodes = apisixv1.UpstreamNodes{}
				return ups, nil
			}
		} else {
			return nil, &TranslateError{
				Field:  "ApisixUpstream",
				Reason: err.Error(),
			}
		}
	}
	var labels types.Labels
	if subset != "" {
		for _, ss := range au.V2beta3().Spec.Subsets {
			if ss.Name == subset {
				labels = ss.Labels
				break
			}
		}
	}
	// Filter nodes by subset.
	nodes, err := t.TranslateEndpoint(*ep, port, labels)
	if err != nil {
		return nil, err
	}
	if au == nil || au.V2beta3().Spec == nil {
		ups.Nodes = nodes
		return ups, nil
	}

	upsCfg := &au.V2beta3().Spec.ApisixUpstreamConfig
	for _, pls := range au.V2beta3().Spec.PortLevelSettings {
		if pls.Port == port {
			upsCfg = &pls.ApisixUpstreamConfig
			break
		}
	}
	ups, err = t.TranslateUpstreamConfigV2beta3(upsCfg)
	if err != nil {
		return nil, err
	}
	ups.Nodes = nodes
	return ups, nil
}

func (t *translator) TranslateEndpoint(endpoint kube.Endpoint, port int32, labels types.Labels) (apisixv1.UpstreamNodes, error) {
	namespace, err := endpoint.Namespace()
	if err != nil {
		log.Errorw("failed to get endpoint namespace",
			zap.Error(err),
			zap.Any("endpoint", endpoint),
		)
		return nil, err
	}
	svcName := endpoint.ServiceName()
	svc, err := t.ServiceLister.Services(namespace).Get(svcName)
	if err != nil {
		return nil, &TranslateError{
			Field:  "service",
			Reason: err.Error(),
		}
	}

	var svcPort *corev1.ServicePort
	for _, exposePort := range svc.Spec.Ports {
		if exposePort.Port == port {
			svcPort = &exposePort
			break
		}
	}
	if svcPort == nil {
		return nil, &TranslateError{
			Field:  "service.spec.ports",
			Reason: "port not defined",
		}
	}
	// As nodes is not optional, here we create an empty slice,
	// not a nil slice.
	nodes := make(apisixv1.UpstreamNodes, 0)
	for _, hostport := range endpoint.Endpoints(svcPort) {
		nodes = append(nodes, apisixv1.UpstreamNode{
			Host: hostport.Host,
			Port: hostport.Port,
			// FIXME Custom node weight
			Weight: DefaultWeight,
		})
	}
	if labels != nil {
		nodes = t.filterNodesByLabels(nodes, labels, namespace)
		return nodes, nil
	}
	return nodes, nil
}

func (t *translator) filterNodesByLabels(nodes apisixv1.UpstreamNodes, labels types.Labels, namespace string) apisixv1.UpstreamNodes {
	if labels == nil {
		return nodes
	}

	filteredNodes := make(apisixv1.UpstreamNodes, 0)
	for _, node := range nodes {
		podName, err := t.PodProvider.GetPodCache().GetNameByIP(node.Host)
		if err != nil {
			log.Errorw("failed to find pod name by ip, ignore it",
				zap.Error(err),
				zap.String("pod_ip", node.Host),
			)
			continue
		}
		pod, err := t.PodLister.Pods(namespace).Get(podName)
		if err != nil {
			log.Errorw("failed to find pod, ignore it",
				zap.Error(err),
				zap.String("pod_name", podName),
			)
			continue
		}
		if labels.IsSubsetOf(pod.Labels) {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes
}
