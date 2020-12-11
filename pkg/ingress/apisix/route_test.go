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

import "testing"

func TestApisixRoute_Convert(t *testing.T) {

}

var routeYaml = `
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  annotations:
    k8s.apisix.apache.org/cors-allow-headers: DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization,openID,audiotoken
    k8s.apisix.apache.org/cors-allow-methods: HEAD,GET,POST,PUT,PATCH,DELETE
    k8s.apisix.apache.org/cors-allow-origin: '*'
    k8s.apisix.apache.org/enable-cors: "true"
    k8s.apisix.apache.org/ssl-redirect: "false"
    k8s.apisix.apache.org/whitelist-source-range: 58.210.212.110,10.244.0.0/16
  name: httpserver-route
spec:
  rules:
  - host: test1.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: api6
          servicePort: 80
        path: /test*
        plugins:
        - config:
            key: apisix-chash-key
            uri_args:
            - pId
            - userId|device
          enable: true
          name: aispeech-chash
      - backend:
          serviceName: httpserver
          servicePort: 8080
        path: /hello*
        plugins:
        - config:
            key: apisix-chash-key
            uri_args:
            - productId2
            - productId|deviceName
          enable: true
          name: aispeech-chash
`
