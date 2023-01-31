// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"fmt"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateApisixConsumerV2beta3(ac *configv2beta3.ApisixConsumer) (*apisixv1.Consumer, error) {
	// As the CRD schema ensures that only one authN can be configured,
	// so here the order is no matter.

	plugins := make(apisixv1.Plugins)
	if ac.Spec.AuthParameter.KeyAuth != nil {
		cfg, err := t.translateConsumerKeyAuthPluginV2beta3(ac.Namespace, ac.Spec.AuthParameter.KeyAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid key auth config: %s", err)
		}
		plugins["key-auth"] = cfg
	} else if ac.Spec.AuthParameter.BasicAuth != nil {
		cfg, err := t.translateConsumerBasicAuthPluginV2beta3(ac.Namespace, ac.Spec.AuthParameter.BasicAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid basic auth config: %s", err)
		}
		plugins["basic-auth"] = cfg
	} else if ac.Spec.AuthParameter.JwtAuth != nil {
		cfg, err := t.translateConsumerJwtAuthPluginV2beta3(ac.Namespace, ac.Spec.AuthParameter.JwtAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid jwt auth config: %s", err)
		}
		plugins["jwt-auth"] = cfg
	} else if ac.Spec.AuthParameter.WolfRBAC != nil {
		cfg, err := t.translateConsumerWolfRBACPluginV2beta3(ac.Namespace, ac.Spec.AuthParameter.WolfRBAC)
		if err != nil {
			return nil, fmt.Errorf("invalid wolf rbac config: %s", err)
		}
		plugins["wolf-rbac"] = cfg
	} else if ac.Spec.AuthParameter.HMACAuth != nil {
		cfg, err := t.translateConsumerHMACAuthPluginV2beta3(ac.Namespace, ac.Spec.AuthParameter.HMACAuth)
		if err != nil {
			return nil, fmt.Errorf("invaild hmac auth config: %s", err)
		}
		plugins["hmac-auth"] = cfg
	}

	consumer := apisixv1.NewDefaultConsumer()
	consumer.Username = apisixv1.ComposeConsumerName(ac.Namespace, ac.Name)
	consumer.Plugins = plugins
	return consumer, nil
}

func (t *translator) TranslateApisixConsumerV2(ac *configv2.ApisixConsumer) (*apisixv1.Consumer, error) {
	// As the CRD schema ensures that only one authN can be configured,
	// so here the order is no matter.

	plugins := make(apisixv1.Plugins)
	if ac.Spec.AuthParameter.KeyAuth != nil {
		cfg, err := t.translateConsumerKeyAuthPluginV2(ac.Namespace, ac.Spec.AuthParameter.KeyAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid key auth config: %s", err)
		}
		plugins["key-auth"] = cfg
	} else if ac.Spec.AuthParameter.BasicAuth != nil {
		cfg, err := t.translateConsumerBasicAuthPluginV2(ac.Namespace, ac.Spec.AuthParameter.BasicAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid basic auth config: %s", err)
		}
		plugins["basic-auth"] = cfg
	} else if ac.Spec.AuthParameter.JwtAuth != nil {
		cfg, err := t.translateConsumerJwtAuthPluginV2(ac.Namespace, ac.Spec.AuthParameter.JwtAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid jwt auth config: %s", err)
		}
		plugins["jwt-auth"] = cfg
	} else if ac.Spec.AuthParameter.WolfRBAC != nil {
		cfg, err := t.translateConsumerWolfRBACPluginV2(ac.Namespace, ac.Spec.AuthParameter.WolfRBAC)
		if err != nil {
			return nil, fmt.Errorf("invalid wolf rbac config: %s", err)
		}
		plugins["wolf-rbac"] = cfg
	} else if ac.Spec.AuthParameter.HMACAuth != nil {
		cfg, err := t.translateConsumerHMACAuthPluginV2(ac.Namespace, ac.Spec.AuthParameter.HMACAuth)
		if err != nil {
			return nil, fmt.Errorf("invaild hmac auth config: %s", err)
		}
		plugins["hmac-auth"] = cfg
	} else if ac.Spec.AuthParameter.LDAPAuth != nil {
		cfg, err := t.translateConsumerLDAPAuthPluginV2(ac.Namespace, ac.Spec.AuthParameter.LDAPAuth)
		if err != nil {
			return nil, fmt.Errorf("invalid ldap auth config: %s", err)
		}
		plugins["ldap-auth"] = cfg
	}

	consumer := apisixv1.NewDefaultConsumer()
	consumer.Username = apisixv1.ComposeConsumerName(ac.Namespace, ac.Name)
	consumer.Plugins = plugins
	return consumer, nil
}
