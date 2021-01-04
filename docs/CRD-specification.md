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

Use the following CRDs to define your routing rules. Normally, you only need to use ApisixRoute to complete a simple route definition.

## CRD Types

- [ApisixRoute](#apisixroute)
- [ApisixService](#apisixservice)
- [ApisixUpstream](#apisixupstream)
- [ApisixTls](#apisixtls)

## ApisixRoute

`ApisixRoute` corresponds to the `Route` object in Apache APISIX. The `Route` matches the client's request by defining rules,
then loads and executes the corresponding plugin based on the matching result, and forwards the request to the specified Upstream.
To learn more, please check the [Apache APISIX architecture-design docs](https://github.com/apache/apisix/blob/master/doc/architecture-design.md#route).

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
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
          - name: limit-count
            enable: true
            config:
              count: 2
              time_window: 60
              rejected_code: 503
              key: remote_addr
```

|     Field     |  Type    |                    Description                     |
|---------------|----------|----------------------------------------------------|
| rules         | array    | ApisixRoute's request matching rules.              |
| host          | string   | The requested host.                                |
| http          | string   | The routing rule of the request.                   |
| paths         | array    | The routing rule of the request.                   |
| backend       | object   | Backend service information configuration.         |
| serviceName   | string   | The name of backend service. `namespace + serviceName + servicePort` form a unique identifier to match the back-end service.                      |
| servicePort   | int      | The port of backend service. `namespace + serviceName + servicePort` form a unique identifier to match the back-end service.                      |
| path          | string   | The URI matched by the route. Supports exact match and prefix match. Example，exact match: `/hello`, prefix match: `/hello*`.                     |
| plugins       | array    | Custom plugin collection (Plugins defined in the `route` level). For more plugin information, please refer to the [Apache APISIX plugin docs](https://github.com/apache/apisix/tree/master/doc/plugins).    |
| name          | string   | The name of the plugin. For more information about the example plugin, please check the [limit-count docs](https://github.com/apache/apisix/blob/master/doc/plugins/limit-count.md#Attributes).             |
| enable        | boolean  | Whether to enable the plugin, `true`: means enable, `false`: means disable.      |
| config        | object   | Configuration of plugin information. Note: The check of configuration schema is missing now, so please be careful when editing.    |

**Support partial `annotation`**

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  annotations:
    k8s.apisix.apache.org/ingress.class: apisix_group
    k8s.apisix.apache.org/ssl-redirect: 'false'
    k8s.apisix.apache.org/whitelist-source-range:
      - 1.2.3.4/16
      - 4.3.2.1/8
  name: httpserverRoute
  namespace: cloud
spec:
```

|         Field                                  |    Type    |                       Description                                  |
|------------------------------------------------|------------|--------------------------------------------------------------------|
| `k8s.apisix.apache.org/ssl-redirect`           | boolean    | Whether to use SSL forwarding, true: means using SSL forwarding, false: means not using SSL forwarding.   |
| `k8s.apisix.apache.org/ingress.class`          | string     | Grouping of ingress.                                               |
| `k8s.apisix.apache.org/whitelist-source-range` | array      | Whitelist of IPs allowed to be accessed.                           |

## ApisixService

`ApisixService` corresponds to the `Service` object in Apache APISIX.
A `Service` is an abstraction of an API (which can also be understood as a set of Route abstractions). It usually corresponds to the upstream service abstraction. Between `Route` and `Service`, usually the relationship of N:1.
To learn more, please check the [Apache APISIX architecture-design docs](https://github.com/apache/apisix/blob/master/doc/architecture-design.md#service).

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
    - name: limit-count
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
| plugins       | array   | Custom plugin collection (Plugins defined in the `service` level). For more plugin information, please refer to the [Apache APISIX plugins docs](https://github.com/apache/apisix/tree/master/doc/plugins). |
| name          | string   | The name of the plugin. For more information about the example plugin, please check the [limit-count docs](https://github.com/apache/apisix/blob/master/doc/plugins/limit-count.md#Attributes).    |
| enable        | boolean  | Whether to enable the plugin, `true`: means enable, `false`: means disable.      |
| config        | object   | Configuration of plugin information. Note: The check of configuration schema is missing now, so please be careful when editing.    |

## ApisixUpstream

`ApisixUpstream` corresponds to the `Upstream` object in Apache APISIX.
Upstream is a virtual host abstraction that performs load balancing on a given set of service nodes according to configuration rules.
Upstream address information can be directly configured to `Route` (or `Service`). When Upstream has duplicates, you need to use "reference" to avoid duplication.
To learn more, please check the [Apache APISIX architecture-design docs](https://github.com/apache/apisix/blob/master/doc/architecture-design.md#upstream).

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
| loadbalancer  | string/object   | The load balance algorithm of this upstream service, optional value can be `roundrobin` or `chash`.  |

## ApisixTls

`ApisixTls` corresponds to the `SSL` of the `Router` object in Apache APISIX.
`SSL` loads the matching route. (Default) Use SNI (Server Name Indication) as the primary index.
To learn more, please check the [Apache APISIX architecture-design docs](https://github.com/apache/apisix/blob/master/doc/architecture-design.md#router).

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
| hosts         | array    | Host of `SNI`.                  |
| secret        | object   | The secret of kubernetes, it works with `ApisixTls`.            |
| name          | string   | The name of `secret`. `namespace` and `name` are the unique identifier to match kubernetes secret.       |
| namespace     | string   | The namespace of `secret`. `namespace` and `name` are the unique identifier to match kubernetes secret.  |

[Back to top](#crd-types)
