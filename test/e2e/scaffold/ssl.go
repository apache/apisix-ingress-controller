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

package scaffold

import (
	"encoding/base64"
	"fmt"

	"github.com/gruntwork-io/terratest/modules/k8s"
)

const (
	_secretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  name: %s
data:
  cert: %s
  key: %s
`
	_api6tlsTemplate = `
apiVersion: apisix.apache.org/v1
kind: ApisixTls
metadata:
  name: %s
spec:
  hosts:
  - api6.com
  secret:
    name: %s
    namespace: %s
`
)

func (s *Scaffold) NewSecret(name, cert, key string) error {
	certBase64 := base64.StdEncoding.EncodeToString([]byte(cert))
	keyBase64 := base64.StdEncoding.EncodeToString([]byte(key))
	secret := fmt.Sprintf(_secretTemplate, name, certBase64, keyBase64)
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, secret); err != nil {
		return err
	}
	return nil
}

func (s *Scaffold) NewApisixTls(name, secretName string) error {
	tls := fmt.Sprintf(_api6tlsTemplate, name, secretName, s.kubectlOptions.Namespace)
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, tls); err != nil {
		return err
	}
	return nil
}
