---
title: Enable authentication and restriction
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

Consumers are used for the authentication method controlled by Apache APISIX, if users want to use their own auth system or 3rd party systems, use OIDC.  

## Attributes

### Authentication

#### `keyAuth`

Consumers add their key either in a header `apikey` to authenticate their requests.

```yaml
keyAuth:
  value:
    key: ${key}
```

#### `basicAuth`

Consumers add their key either in a header `Authentication` to authenticate their requests.

```yaml
basicAuth:
  value:
    username: ${username}
    password: ${password}
```

### Restriction

#### `whitelist` or `blacklist`

`whitelist`: Grant full access to all users specified in the provided list, **has the priority over `allowed_by_methods`**  
`blacklist`: Reject connection to all users specified in the provided list, **has the priority over `whitelist`**

```yaml
plugins:
- name: consumer-restriction
  enable: true
  config:
    blacklist:
    - "${consumer_name}"
    - "${consumer_name}"
```

#### `allowed_by_methods`

HTTP methods can be `methods:["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE", "PURGE"]`

```yaml
plugins:
- name: consumer-restriction
  enable: true
  config:
    allowed_by_methods:
    - user: "${consumer_name}"
      methods:
      - "POST"
      - "GET"
    - user: "${consumer_name}"
      methods:
      - "GET"
```

## Example

### Prepare env

Kubernetes cluster:

1. [apisix-ingress-controller](../deployments/minikube.md).
2. httpbin.

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

Requests from foo:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:foo-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

### How to enable `Restriction`

We can also use the `consumer-restriction` Plugin to restrict our user from accessing the API.

#### How to restrict `consumer_name`

The following is an example. The `consumer-restriction` plugin is enabled on the specified route to restrict `consumer_name` access.

* **consumer_name**: Add the `username` of `consumer` to a whitelist or blacklist (supporting single or multiple consumers) to restrict access to services or routes.

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

How to get `default_jack1`:

> view ApisixConsumer resource object from this namespace `default`
>
> ```shell
> $ kubectl get apisixconsumers.apisix.apache.org -n default
> NAME    AGE
> foo     14h
> jack1   14h
> jack2   14h
> ```
>
> `${consumer_name}` = `${namespace}_${ApisixConsumer_name}` --> `default_foo`  
> `${consumer_name}` = `${namespace}_${ApisixConsumer_name}` --> `default_jack1`  
> `${consumer_name}` = `${namespace}_${ApisixConsumer_name}` --> `default_jack2`

**Example usage**

Requests from jack1:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack1-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

Requests from jack2:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack2-key' -i
```

```shell
HTTP/1.1 403 Forbidden
...
{"message":"The consumer_name is forbidden."}
```

#### How to restrict `allowed_by_methods`

This example restrict the user `jack2` to only `GET` on the resource.

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

**Example usage**

Requests from jack1:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack1-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack1-key' -d '' -i
```

```shell
HTTP/1.1 200 OK
...
```

Requests from jack2:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack2-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -H 'apikey:jack2-key' -d '' -i
```

```shell
HTTP/1.1 403 Forbidden
...
```

### Disable authentication and restriction

To disable the `consumer-restriction` Plugin, you can set the `enable: false` from the `plugins` configuration.  
Also, disable the `keyAuth`, you can set the `enable: false` from the `authentication` configuration.

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
     enable: false
     type: keyAuth
   plugins:
   - name: consumer-restriction
     enable: false
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

Requests:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: local.httpbin.org' -i
```

```shell
HTTP/1.1 200 OK
...
```
