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
package apisix

import (
	ingress "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ingressConf "github.com/api7/ingress-controller/pkg/kube"
	"github.com/api7/ingress-controller/pkg/seven/conf"
	apisix "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

const (
	ApisixTls = "ApisixTls"
)

type ApisixTlsCRD ingress.ApisixTls

// Convert convert to  apisix.Ssl from ingress.ApisixTls CRD
func (as *ApisixTlsCRD) Convert(sc Secreter) (*apisix.Ssl, error) {
	name := as.Name
	namespace := as.Namespace
	_, group := BuildAnnotation(as.Annotations)
	conf.AddGroup(group)

	id := namespace + "_" + name
	secretName := as.Spec.Secret.Name
	secretNamespace := as.Spec.Secret.Namespace
	secret, err := sc.FindByName(secretNamespace, secretName)
	if err != nil {
		return nil, err
	}
	cert := string(secret.Data["cert"])
	key := string(secret.Data["key"])
	status := 1
	snis := make([]*string, 0)
	for _, host := range as.Spec.Hosts {
		snis = append(snis, &host)
	}
	ssl := &apisix.Ssl{
		ID:     &id,
		Snis:   snis,
		Cert:   &cert,
		Key:    &key,
		Status: &status,
		Group:  &group,
	}
	return ssl, nil
}

type Secreter interface {
	FindByName(namespace, name string) (*v1.Secret, error)
}

type SecretClient struct{}

func (sc *SecretClient) FindByName(namespace, name string) (*v1.Secret, error) {
	clientSet := ingressConf.GetKubeClient()
	return clientSet.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
}
