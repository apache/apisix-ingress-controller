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

#### Key Auth

Consumers add their key either in a header or query string parameter to authenticate their requests. For more information about `Key Auth`, please refer to [APISIX key-auth plugin](https://apisix.apache.org/docs/apisix/plugins/key-auth/).
Also, we can using the `secretRef` field to reference a K8s Secret object so that we can avoid the hardcoded sensitive data in the ApisixConsumer object. For reference Secret use example, please refer to the [key-auth-reference-secret-object](#key-auth-reference-secret-object).

<details>
  <summary>Key Auth yaml configure</summary>

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: ${name}
spec:
  authParameter:
    keyAuth:
      value:
        key: ${key} #required
```

</details>

#### Basic Auth

Consumers add their key in a header to authenticate their requests. For more information about `Basic Auth`, please refer to [APISIX basic-auth plugin](https://apisix.apache.org/docs/apisix/plugins/basic-auth/).
Also, we can using the `secretRef` field to reference a K8s Secret object so that we can avoid the hardcoded sensitive data in the ApisixConsumer object. For reference Secret use example, please refer to the [key-auth-reference-secret-object](#key-auth-reference-secret-object).

<details>
  <summary>Basic Auth yaml configure</summary>

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: ${name}
spec:
  authParameter:
    basicAuth:
      value:
        username: ${username} #required
        password: ${password} #required
```

</details>

#### JWT Auth

The consumer then adds its key to the query string parameter, request header, or cookie to verify its request. For more information about `JWT Auth`, please refer to [APISIX jwt-auth plugin](https://apisix.apache.org/docs/apisix/plugins/jwt-auth/).
Also, we can using the `secretRef` field to reference a K8s Secret object so that we can avoid the hardcoded sensitive data in the ApisixConsumer object. For reference Secret use example, please refer to the [key-auth-reference-secret-object](#key-auth-reference-secret-object).

:::note Need to expose API
This plugin will add `/apisix/plugin/jwt/sign` to sign. You may need to use `public-api` plugin to expose it.
:::

<details>
  <summary>JWT Auth yaml configure</summary>

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: ${name}
spec:
  authParameter:
    wolfRbac:
      value:
        key: "${key}"                                    #required
        secret: "${secret}"                              #optional
        public_key: "${public_key}"                      #optional, required when algorithm attribute selects RS256 algorithm.
        private_key: "{private_key}"                     #optional, required when algorithm attribute selects RS256 algorithm.
        algorithm: "${HS256 | HS512 | RS256}"            #optional
        exp: ${ 86400 | token's expire time, in seconds} #optional
        algorithm: ${true | false}                       #optional
```

</details>

#### `Wolf RBAC`

To use wolfRbac authentication, you need to start and install [wolf-server](https://github.com/iGeeky/wolf/blob/master/quick-start-with-docker/README.md). For more information about `Wolf RBAC`, please refer to [APISIX wolf-rbac plugin](https://apisix.apache.org/zh/docs/apisix/plugins/wolf-rbac/).
Also, we can using the `secretRef` field to reference a K8s Secret object so that we can avoid the hardcoded sensitive data in the ApisixConsumer object. For reference Secret use example, please refer to the [key-auth-reference-secret-object](#key-auth-reference-secret-object).

:::note This plugin will add several APIs

* /apisix/plugin/wolf-rbac/login
* /apisix/plugin/wolf-rbac/change_pwd
* /apisix/plugin/wolf-rbac/user_info

You may need to use `public-api` plugin to expose it.
:::

<details>
  <summary>Wolf RBAC yaml configure</summary>

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: ${name}
spec:
  authParameter:
    wolfRBAC:
      value:
      server: "${server of wolf-rbac}"                            #optional
      appid: "${appid of wolf-rbac}"                              #optional
      header_prefix: "${X- | X-UserId | X-Username | X-Nickname}" #optional
```

</details>

### [Restriction](https://apisix.apache.org/docs/apisix/plugins/consumer-restriction/)

#### `whitelist` or `blacklist`

`whitelist`: Grant full access to all users specified in the provided list, **has the priority over `allowed_by_methods`**
`blacklist`: Reject connection to all users specified in the provided list, **has the priority over `whitelist`**

<details>
  <summary>whitelist or blacklist with consumer-restriction yaml configure</summary>

```yaml
plugins:
- name: consumer-restriction
  enable: true
  config:
    blacklist:
    - "${consumer_name}"
    - "${consumer_name}"
```

</details>

#### `allowed_by_methods`

HTTP methods can be `methods:["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE", "PURGE"]`

<details>
  <summary>allowed_by_methods with consumer-restriction yaml configure</summary>

```yaml
plugins:
- name: consumer-restriction
  enable: true
  config:
    allowed_by_methods:
    - user: "${consumer_name}"
      methods:
      - "${GET | POST | PUT |...}"
      - "${GET | POST | PUT |...}"
    - user: "${consumer_name}"
      methods:
      - "${GET | POST | PUT |...}"
```

</details>

## Example

[Refer to the corresponding e2e test case.](../../../../test/e2e/suite-plugins/suite-plugins-authentication/)

### Prepare env

To use this tutorial, you must deploy `Ingress APISIX` and `httpbin` in Kubernetes cluster.

* Installing [`Ingress APISIX`](../deployments/minikube.md).
* Deploy `httpbin` service.

```shell
#Now, try to deploy httpbin to your Kubernetes cluster:
kubectl run httpbin --image kennethreitz/httpbin --port 80
kubectl expose pod httpbin --port 80
```

### How to enable `Authentication`

#### Enable `keyAuth`

The following is an example. The `keyAuth` is enabled on the specified route to restrict user access.

* Creates an ApisixConsumer, and set the attributes of plugin `key-auth`:

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

* Creates an ApisixRoute, and enable plugin `key-auth`:

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
      - httpbin.org
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

* Requests from foo:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'apikey:foo-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

##### Key Auth reference Secret object

<details>
  <summary>ApisixRoute with keyAuth consumer using secret example</summary>

* Creates a `Secret` object:

```shell
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: foovalue
data:
  key: Zm9vLWtleQ==
EOF
```

* Creates an ApisixConsumer and reference `Secret` object:

```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: foo
spec:
  authParameter:
    keyAuth:
      secretRef:
        name: foovalue
EOF
```

* Creates an ApisixRoute, and enables plugin `key-auth`:

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
      - httpbin.org
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

* Requests from foo:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'apikey:foo-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

</details>

#### Enable `JWT Auth`

* Creates an ApisixConsumer, and set the attributes of plugin `jwt-auth`:

```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: foo2
spec:
  authParameter:
    jwtAuth:
      value:
        key: foo2-key
EOF
```

* Use the `public-api` plugin to expose the public API:

```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: default
spec:
  http:
  - name: public-api
    match:
      paths:
      - /apisix/plugin/jwt/sign
    backends:
    - serviceName: apisix-admin
      servicePort: 9180
    plugins:
    - name: public-api
      enable: true
EOF
```

* Creates an ApisixRoute, and enable the jwt-auth plugin:

```shell
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /*
   backends:
   - serviceName: httpbin
     servicePort: 80
   authentication:
     enable: true
     type: jwtAuth
EOF
```

* Get the token:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/apisix/plugin/jwt/sign?key=foo2-key -H 'Host: httpbin.org' -i
```

```shell
HTTP/1.1 200 OK
...
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJrZXkiOiJ1c2VyLWtleSIsImV4cCI6MTU2NDA1MDgxMX0.Us8zh_4VjJXF-TmR5f8cif8mBU7SuefPlpxhH0jbPVI
```

* Without token:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -i
```

```shell
HTTP/1.1 401
...
{"message":"Missing JWT token in request"}
```

* Request header with token:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJrZXkiOiJ1c2VyLWtleSIsImV4cCI6MTU2NDA1MDgxMX0.Us8zh_4VjJXF-TmR5f8cif8mBU7SuefPlpxhH0jbPVI' -i
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

* Create ApisixConsumer jack1:

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

* Create ApisixConsumer jack2:

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

* Creates an ApisixRoute, and enable config `whitelist`  of the plugin `consumer-restriction`:

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
     - httpbin.org
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

:::note The `default_jack1` generation rules:

view ApisixConsumer resource object from this namespace `default`

```shell
$ kubectl get apisixconsumers.apisix.apache.org -n default
NAME    AGE
foo     14h
jack1   14h
jack2   14h
```

`${consumer_name}` = `${namespace}_${ApisixConsumer_name}` --> `default_foo`
`${consumer_name}` = `${namespace}_${ApisixConsumer_name}` --> `default_jack1`
`${consumer_name}` = `${namespace}_${ApisixConsumer_name}` --> `default_jack2`

:::

**Example usage**

* Requests from jack1:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'apikey:jack1-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

* Requests from jack2:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'apikey:jack2-key' -i
```

```shell
HTTP/1.1 403 Forbidden
...
{"message":"The consumer_name is forbidden."}
```

#### How to restrict `allowed_by_methods`

This example restrict the user `jack2` to only `GET` on the resource.

* Creates an ApisixRoute, and enable config `allowed_by_methods`  of the plugin `consumer-restriction`:

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
     - httpbin.org
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

* Requests from jack1:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'apikey:jack1-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'apikey:jack1-key' -d '' -i
```

```shell
HTTP/1.1 200 OK
...
```

* Requests from jack2:

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'apikey:jack2-key' -i
```

```shell
HTTP/1.1 200 OK
...
```

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX} -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -H 'apikey:jack2-key' -d '' -i
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
     - httpbin.org
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

```shell
kubectl  exec -it -n ${namespace of Apache APISIX} ${pod of Apache APISIX}  -- curl http://127.0.0.1:9080/anything -H 'Host: httpbin.org' -i
```

```shell
HTTP/1.1 200 OK
...
```
