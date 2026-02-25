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

package translator

import (
	"github.com/go-logr/logr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

type Translator struct {
	Log                   logr.Logger
	ListenerPortMatchMode config.ListenerPortMatchMode
}

func normalizeMode(mode config.ListenerPortMatchMode) config.ListenerPortMatchMode {
	switch mode {
	case "", config.ListenerPortMatchModeAuto:
		return config.ListenerPortMatchModeAuto
	case config.ListenerPortMatchModeExplicit, config.ListenerPortMatchModeOff:
		return mode
	default:
		return config.ListenerPortMatchModeAuto
	}
}

func NewTranslator(log logr.Logger, mode config.ListenerPortMatchMode) *Translator {
	return &Translator{
		Log:                   log.WithName("translator"),
		ListenerPortMatchMode: normalizeMode(mode),
	}
}

func hasExplicitListenerTarget(parentRefs []gatewayv1.ParentReference) bool {
	for _, parentRef := range parentRefs {
		// Skip non-Gateway parentRefs (e.g. GAMMA Service mesh refs) — they
		// are not relevant to listener port injection.
		if parentRef.Kind != nil && *parentRef.Kind != "Gateway" {
			continue
		}
		if parentRef.SectionName != nil && *parentRef.SectionName != "" {
			return true
		}
		if parentRef.Port != nil {
			return true
		}
	}

	return false
}

func (t *Translator) shouldInjectServerPortVars(parentRefs []gatewayv1.ParentReference, ports map[int32]struct{}) bool {
	if len(ports) == 0 {
		return false
	}

	explicit := hasExplicitListenerTarget(parentRefs)

	switch t.ListenerPortMatchMode {
	case config.ListenerPortMatchModeOff:
		if explicit {
			t.Log.V(1).Info("listener_port_match_mode is 'off'; ignoring explicit listener targeting", "parent_refs", len(parentRefs))
		}
		return false
	case config.ListenerPortMatchModeExplicit:
		return explicit
	case config.ListenerPortMatchModeAuto:
		return explicit || len(ports) > 1
	default:
		return explicit || len(ports) > 1
	}
}

type TranslateResult struct {
	Services       []*adctypes.Service
	SSL            []*adctypes.SSL
	GlobalRules    adctypes.GlobalRule
	PluginMetadata adctypes.PluginMetadata
	Consumers      []*adctypes.Consumer
}
