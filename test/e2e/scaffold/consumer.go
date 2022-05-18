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

import "fmt"

var (
	_apisixConsumerBasicAuth = `
apiVersion: %s
kind: ApisixConsumer
metadata:
  name: %s
spec:
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
    authParameter:
      keyAuth:
        secretRef:
          name: %s
  `
)

func (s *Scaffold) ApisixConsumerBasicAuthCreated(name, username, password string) error {
	ac := fmt.Sprintf(_apisixConsumerBasicAuth, s.opts.APISIXConsumerVersion, name, username, password)
	return s.CreateResourceFromString(ac)
}

func (s *Scaffold) ApisixConsumerBasicAuthSecretCreated(name, secret string) error {
	ac := fmt.Sprintf(_apisixConsumerBasicAuthSecret, s.opts.APISIXConsumerVersion, name, secret)
	return s.CreateResourceFromString(ac)
}

func (s *Scaffold) ApisixConsumerKeyAuthCreated(name, key string) error {
	ac := fmt.Sprintf(_apisixConsumerKeyAuth, s.opts.APISIXConsumerVersion, name, key)
	return s.CreateResourceFromString(ac)
}

func (s *Scaffold) ApisixConsumerKeyAuthSecretCreated(name, secret string) error {
	ac := fmt.Sprintf(_apisixConsumerKeyAuthSecret, s.opts.APISIXConsumerVersion, name, secret)
	return s.CreateResourceFromString(ac)
}
