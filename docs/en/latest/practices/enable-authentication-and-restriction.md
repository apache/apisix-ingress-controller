---
title: enable authentication and restriction
---

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

## Description

Consumers are useful when you have different consumers requesting the same API and you need to execute different Plugin and Upstream configurations based on the consumer. These need to be used in conjunction with the user authentication system.  
## Attributes
* Authentication
  * `basicAuth`
  * `keyAuth`
* Restriction
  * `consumer_name`
  * `allowed_by_methods`
## Example

### Prepare env
Kubernetes cluster: 
1. [apisix-ingress-controller](https://apisix.apache.org/docs/ingress-controller/deployments/minikube/)
2. httpbin
```shell
#Now, try to deploy it to your Kubernetes cluster:
kubectl run httpbin --image kennethreitz/httpbin --port 80
kubectl expose pod httpbin --port 80
```

### How to enable `Authentication`

The following is an example. The `keyAuth` is enabled on the specified route to restrict user access.  

Create ApisixConsumer foo:

```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: foo
spec:
  authParameter:
    keyAuth:
      value:
        key: foo-key
EOF
```

ApisixRoute:

```shell
kubectl apply -f -<<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpserver-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - local.httpbin.org
      paths:
      - /*
    backends:
    - serviceName: httpbin
      servicePort: 80
    authentication:
      enable: true
      type: keyAuth
EOF
```

**Test keyAuth**

Requests from foo:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:foo-key' -v

HTTP/1.1 200 OK
...
```

### How to enable `Restriction`
We can also use the `consumer-restriction` Plugin to restrict our user from accessing the API. 

The configure:
> Use `whitelist` or `blacklist` restrict `consumer_name` 
> ```yaml
> config:
>   whitelist:
>   - "${namespace}_${name:1}"
>     "${namespace}_${name:...}"
> ```
> Restrict `allowed_by_methods`
> ```yaml
>config:
>  allowed_by_methods:
>  - user: "${namespace}_${name:1}"
>    methods:
>    - "$(method)[GET,POST]"
>  - user: "${namespace}_${name:...}"
>    methods:
>    - "$(method)[GET,POST]"
> ```
> &nbsp; 

#### How to restrict `consumer_name`

The following is an example. The `consumer-restriction` plugin is enabled on the specified route to restrict consumer access.

Create ApisixConsumer jack1:
```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: jack1
spec:
  authParameter:
    keyAuth:
      value:
        key: jack1-key
EOF
```

Create ApisixConsumer jack2:
```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: jack2
spec:
  authParameter:
    keyAuth:
      value:
        key: jack2-key
EOF
``` 

ApisixRoute:
```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
 name: httpserver-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - local.httpbin.org
     paths:
       - /*
   backends:
   - serviceName: httpbin
     servicePort: 80
   authentication:
     enable: true
     type: keyAuth
   plugins:
   - name: consumer-restriction
     enable: true
     config:
       whitelist:
       - "default_jack1"
EOF
```

**Test Plugin**

Requests from jack1:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack1-key' -v

HTTP/1.1 200 OK
...
```

Requests from jack2:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack2-key'
HTTP/1.1 403 Forbidden
...
{"message":"The consumer_name is forbidden."}
```

#### How to restrict `allowed_by_methods`

This example restrict the user `jack2` to only `GET` on the resource 

```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
 name: httpserver-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - local.httpbin.org
     paths:
       - /*
   backends:
   - serviceName: httpbin
     servicePort: 80
   authentication:
     enable: true
     type: keyAuth
   plugins:
   - name: consumer-restriction
     enable: true
     config:
       allowed_by_methods:
       - user: "default_jack1"
         methods:
         - "POST"
         - "GET"
       - user: "default_jack2"
         methods:
         - "GET"
EOF
```

**Test Plugin**

Requests from jack1:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack1-key' -v

HTTP/1.1 200 OK
...
```

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack1-key' -d '' -v

HTTP/1.1 200 OK
...
```

Requests from jack2:
```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack2-key' -v 

HTTP/1.1 200 OK
...
```

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack2-key' -d '' -v

HTTP/1.1 403 Forbidden
...
```

### Disable Plugin

When you want to disable the `consumer-restriction` plugin, it is very simple,
you can delete the corresponding json configuration in the plugin configuration,
no need to restart the service, it will take effect immediately:

```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
 name: httpserver-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - local.httpbin.org
     paths:
       - /*
   backends:
   - serviceName: httpbin
     servicePort: 80
   authentication:
     enable: true
     type: keyAuth
   plugins:
   - name: consumer-restriction
     enable: false
     config:
EOF
```

The `consumer-restriction` plugin has been disabled now. It works for other plugins.