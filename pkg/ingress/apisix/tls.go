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
	ingressConf "github.com/api7/ingress-controller/conf"
	ingress "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	apisix "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ApisixTls = "ApisixTls"
)

type ApisixTlsCRD ingress.ApisixTls

// Convert convert to  apisix.Service from ingress.ApisixService CRD
func (as *ApisixTlsCRD) Convert() ([]*apisix.Ssl, error) {
	secretName := as.Spec.Secret.Name
	secretNamespace := as.Spec.Secret.Namespace
	clientSet := ingressConf.GetKubeClient()
	secret, err := clientSet.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	cert := string(secret.Data["cert"])
	key := string(secret.Data["key"])
	result := make([]*apisix.Ssl, 0)
	for _, host := range as.Spec.Hosts {
		ssl := &apisix.Ssl{
			Sni:  &host,
			Cert: &cert,
			Key:  &key,
		}
		result = append(result, ssl)
	}
	return result, nil
}
