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
package scaffold

import (
	"fmt"
)

var (
	_apisixConsumerBasicAuth = `
apiVersion: %s
kind: ApisixConsumer
metadata:
  name: %s
spec:
  description: %s
  authParameter:
    basicAuth:
      value:
        username: %s
        password: %s
`
	_apisixConsumerBasicAuthSecret = `
apiVersion: %s
kind: ApisixConsumer
metadata:
  name: %s
spec:
  description: %s
  authParameter:
    basicAuth:
      secretRef:
        name: %s
`
	_apisixConsumerKeyAuth = `
  apiVersion: %s
  kind: ApisixConsumer
  metadata:
    name: %s
  spec:
    description: %s
    authParameter:
      keyAuth:
        value:
          key: %s
  `
	_apisixConsumerKeyAuthSecret = `
  apiVersion: %s
  kind: ApisixConsumer
  metadata:
    name: %s
  spec:
    description: %s
    authParameter:
      keyAuth:
        secretRef:
          name: %s
  `
)

func (s *Scaffold) ApisixConsumerBasicAuthCreated(name, desc, username, password string) error {
	ac := fmt.Sprintf(_apisixConsumerBasicAuth, s.opts.ApisixResourceVersion, name, desc, username, password)
	return s.CreateVersionedApisixResource(ac)
}

func (s *Scaffold) ApisixConsumerBasicAuthSecretCreated(name, desc, secret string) error {
	ac := fmt.Sprintf(_apisixConsumerBasicAuthSecret, s.opts.ApisixResourceVersion, name, desc, secret)
	return s.CreateVersionedApisixResource(ac)
}

func (s *Scaffold) ApisixConsumerKeyAuthCreated(name, desc, key string) error {
	ac := fmt.Sprintf(_apisixConsumerKeyAuth, s.opts.ApisixResourceVersion, name, desc, key)
	return s.CreateVersionedApisixResource(ac)
}

func (s *Scaffold) ApisixConsumerKeyAuthSecretCreated(name, desc, secret string) error {
	ac := fmt.Sprintf(_apisixConsumerKeyAuthSecret, s.opts.ApisixResourceVersion, name, desc, secret)
	return s.CreateVersionedApisixResource(ac)
}
