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

# API specification

In order to be able to use yaml to define the objects required by apisix in k8s, the following structure is defined.

## ApisixRoute

`ApisixRoute` corresponds to the `route` object in apisix.

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
          - httpserver-plugins
          - ...      
```

|     Field     |  Type    | Description    |
|---------------|----------|--------------------------------|
| apiVersion    | string   | apisix.apache.org/v1           |
| kind          | string   | ApisixRoute         |
| name          | string   | httpserverRoute         |
| namespace     | string   | Specify `namespace`, only one `backend` under `namespace` can be configured in the same yaml file. |

## ApisixService

`ApisixService` corresponds to the `service` object in apisix.

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
    - httpserver-plugins
    - ...
```

|     Field     |  Type    | Description    |
|---------------|----------|----------------|
| apiVersion    | string   | apisix.apache.org/v1    |
| kind          | string   | ApisixService           |

## ApisixUpstream

`ApisixUpstream` corresponds to the `upstream` object in apisix.

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: httpserver
  namespace: cloud
spec:
  loadbalancer: roundrobin
  healthcheck:
  	active:
  		...
  	passive:
  		...
```

|     Field     |  Type    | Description    |
|---------------|----------|----------------|
| apiVersion    | string   | apisix.apache.org/v1    |
| kind          | string   | ApisixUpstream          |

## ApisixPlugin

`ApisixPlugin` corresponds to the `plugin` object in apisix.

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixPlugin
metadata:
  name: httpserver-plugins
  namespace: cloud
spec：
  plugins:
  - plugin: limit-conn
  	enable: true
  	config:
  	  key: value
  - plugin: cors
  	enable: true
  	config:
  	  key: value
```

|     Field     |  Type    | Description    |
|---------------|----------|----------------|
| apiVersion    | string   | apisix.apache.org/v1    |
| kind          | string   | ApisixPlugin           |

## ApisixSSL

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
  	all.duiopen.com
```

|     Field     |  Type    | Description    |
|---------------|----------|----------------|
| apiVersion    | string   | apisix.apache.org/v1    |
| kind          | string   | ApisixSSL           |
