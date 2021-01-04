<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->
# CRD specification

In order to control the behavior of the proxy (Apache APISIX), the following CRDs should be defined

## CRD Types

- [ApisixRoute](#apisixroute)
- [ApisixService](#apisixservice)
- [ApisixUpstream](#apisixupstream)
- [ApisixPlugin](#apisixplugin)
- [ApisixTls](#apisixtls)

## ApisixRoute

`ApisixRoute` corresponds to the `route` object in Apache APISIX.

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  annotations:
    k8s.apisix.apache.org/ingress.class: apisix_group
    k8s.apisix.apache.org/ssl-redirect: 'false'
  name: httpserverRoute
  namespace: cloud
spec:
  rules:
  - host: test.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: httpserver
          servicePort: 8080
        path: /hello*
        plugins:
          - name: limit-conn
            enable: true
            config:
              count: 2
              time_window: 60
              rejected_code: 503
              key: remote_addr
```

|     Field     |  Type    | Description                     |
|---------------|----------|--------------------------------|
| rules         | array    | ApisixRoute's request matching rules. |
| host          | string   | The requested host.        |
| http          | string   | The routing rule of the request.         |
| paths         | array    | The routing rule of the request.      |
| backend       | object   | Backend service information configuration.         |
| serviceName   | string   | The name of the service.        |
| servicePort   | int      | The port of the service.        |
| path          | string   | The URI matched by the route. Supports exact match and prefix match. Example，exact match: `/hello`, prefix match: `/hello/*`.   |
| plugins       | array    | Custom plugin collection (Plugins defined in the `route` level).    |
| name          | string   | The name of the plugin.      |
| enable        | boolean  | Whether to enable the plugin, `true`: means enable, `false`: means disable.      |
| config        | object   |  Configuration of plugin information.    |

## ApisixService

`ApisixService` corresponds to the `service` object in Apache APISIX.

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixService
metadata:
  name: httpserver
  namespace: cloud  
spec:
  upstream: httpserver
  port: 8080
  plugins:   
    - name: limit-conn
      enable: true
      config:
        count: 2
        time_window: 60
        rejected_code: 503
        key: remote_addr
```

|     Field     |  Type    | Description    |
|---------------|----------|----------------|
| upstream      | string   | The name of the upstream service.    |
| port          | int      | The port number of the upstream service.    |
| plugins       | array   | Custom plugin collection (Plugins defined in the `service` level).  |

## ApisixUpstream

`ApisixUpstream` corresponds to the `upstream` object in Apache APISIX.

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: httpserver
  namespace: cloud
spec:
  ports:
    - port: 8080
      loadbalancer: roundrobin
```

|     Field     |  Type    | Description    |
|---------------|----------|----------------|
| ports         | array    | Custom upstream collection.   |
| port          | int      | Upstream service port number.    |
| loadbalancer  | string/object   | Select the algorithm of the upstream service, support `roundrobin` and `chash`.  |

## ApisixPlugin

`ApisixPlugin` corresponds to the `plugin` object in Apache APISIX.

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixPlugin
metadata:
  name: httpserver-plugins
  namespace: cloud
spec:
  plugins:
  - name: limit-conn
    enable: true
    config:
      count: 2
      time_window: 60
      rejected_code: 503
      key: remote_addr
  - name: proxy-rewrite
    enable: true
    config:
      regex_uri:
      - '^/(.*)'
      - '/voice-copy-outer-service/$1'
      scheme: http
      host: internalalpha.talkinggenie.com
      enable_websocket: true
```

|     Field     |  Type    | Description                 |
|---------------|----------|-----------------------------|
| plugins       | array   | Custom plugin collection.    |
| name          | string   | The name of the plugin.      |
| enable        | boolean  | Whether to enable the plugin, `true`: means enable, `false`: means disable.  |
| config        | object   |  Configuration of plugin information. |

## ApisixTls

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixSSL
metadata:
  name: duiopen
spec：
  hosts:
  - asr.duiopen.com
  - tts.duiopen.com
  secret:
    name: all.duiopen.com
    namespace: cloud
```

|     Field     |  Type    | Description                     |
|---------------|----------|---------------------------------|
| hosts         | array    | Host of `Tls`.                  |
| secret        | object   | Secret information.             |
| name          | string   | The name of the secret.         |
| namespace     | string   | The namespace of the secret.    |
