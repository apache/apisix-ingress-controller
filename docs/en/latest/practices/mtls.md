---
title: Configuring Mutual Authentication via ApisixTls
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

In this practice, we will use mTLS to protect our exposed ingress APIs.

To learn more about mTLS, please refer to [Mutual authentication](https://en.wikipedia.org/wiki/Mutual_authentication)

## Prerequisites

- an available Kubernetes cluster
- an available APISIX and APISIX Ingress Controller installation

In this guide, we assume that your APISIX is installed in the `apisix` namespace and `ssl` is enabled, which is not enabled by default in the Helm Chart. To enable it, you need to set `gateway.tls.enabled=true` during installation.

Assuming the SSL port is `9443`.

## Deploy httpbin service

We use [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/) as the service image, See its overview page for details.

Deploy it to the default namespace:

```shell
kubectl run httpbin --image kennethreitz/httpbin --port 80
kubectl expose pod httpbin --port 80
```

## Route the traffic

Since SSL is not configured in ApisixRoute, we can use the config similar to the one in practice [Proxy the httpbin service](./proxy-the-httpbin-service.md).

```yaml
# route.yaml
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpserver-route
spec:
  http:
    - name: httpbin
      match:
        hosts:
          - mtls.httpbin.local
        paths:
          - "/*"
      backend:
        serviceName: httpbin
        servicePort: 80
```

Please remember the host field is `mtls.httpbin.local`. It will be the domain we are going to use.

Test it:

```bash
kubectl -n apisix exec -it <APISIX_POD_NAME> -- curl "http://127.0.0.1:9080/ip" -H "Host: mtls.httpbin.local"
```

It should output:

```json
{
  "origin": "127.0.0.1"
}
```

## Certificates

Before configuring SSL, we must have certificates. Certificates often authorized by certificate provider, which also known as Certification Authority (CA).

You can use [OpenSSL](https://en.wikipedia.org/wiki/Openssl) to generate self-signed certificates for testing purposes. Some pre-generated certificates for this guide are [here](./mtls).

- `ca.pem`: The root CA.
- `server.pem` and `server.key`: Server certificate used to enable SSL (https). Contains correct `subjectAltName` matches domain `mtls.httpbin.local`.
- `user.pem` and `user.key`: Client certificate.

To verify them, use commands below:

```bash
openssl verify -CAfile ./ca.pem ./server.pem
openssl verify -CAfile ./ca.pem ./user.pem
```

## Protect the route using SSL

In APISIX Ingress Controller, we use [ApisixTls](../concepts/apisix_tls.md) resource to protect our routes.

ApisixTls requires a secret which field `cert` and `key` contains the certificate and private key.

A secret yaml containing the certificate mentioned above [is here](./mtls/server-secret.yaml). In this guide, we use this as an example.

```bash
kubectl apply -f ./mtls/server-secret.yaml -n default
```

The secret name is `server-secret`, we created it in the `default` namespace. We will reference this secret in `ApisixTls`.

```yaml
# tls.yaml
apiVersion: apisix.apache.org/v1
kind: ApisixTls
metadata:
  name: sample-tls
spec:
  hosts:
    - mtls.httpbin.local
  secret:
    name: server-secret
    namespace: default
```

The `secret` field contains the secret reference.

Please note that the `hosts` field matches our domain `mtls.httpbin.local`.

Apply this yaml, APISIX Ingress Controller will use our certificate to protect the route. Let's test it.

```bash
kubectl -n apisix exec -it <APISIX_POD_NAME> -- curl --resolve 'mtls.httpbin.local:9443:127.0.0.1' "https://mtls.httpbin.local:9443/ip" -k
```

Some major changes here:

- Use `--resolve` parameter to resolve our domain.
  - No `Host` header set explicit.
- We are using `https` and SSL port `9443`.
- Parameter `-k` to allow insecure connections when using SSL. Because our self-signed certificate is not trusted.

Without the domain `mtls.httpbin.local`, the request won't succeed.

You can add parameter `-v` to log the handshake process.

Now, we configured SSL successfully.

## Mutual Authentication

Like `server-secret`, we will create a `client-ca-secret` to store the CA that verify the certificate client presents.

```bash
kubectl apply -f ./mtls/client-ca-secret.yaml -n default
```

Then, change our ApisixTls and apply it:

```yaml
# mtls.yaml
apiVersion: apisix.apache.org/v1
kind: ApisixTls
metadata:
  name: sample-tls
spec:
  hosts:
    - mtls.httpbin.local
  secret:
    name: server-secret
    namespace: default
  client:
    caSecret:
      name: client-ca-secret
      namespace: default
    depth: 10
```

The `client` field references the secret, `depth` indicates the max certificate chain length.

Let's try to connect the route without any chanegs:

```bash
kubectl -n apisix exec -it <APISIX_POD_NAME> -- curl --resolve 'mtls.httpbin.local:9443:127.0.0.1' "https://mtls.httpbin.local:9443/ip" -k
```

If everything works properly, it will return a `400 Bad Request`.

From APISIX access log, we could find logs like this:

```log
2021/05/27 17:20:54 [error] 43#43: *106132 [lua] init.lua:293: http_access_phase(): client certificate was not present, client: 127.0.0.1, server: _, request: "GET /ip HTTP/2.0", host: "mtls.httpbin.local:9443"
127.0.0.1 - - [27/May/2021:17:20:54 +0000] mtls.httpbin.local:9443 "GET /ip HTTP/2.0" 400 154 0.000 "-" "curl/7.76.1" - - - "http://mtls.httpbin.local:9443"
```

That means our mutual authentication has been enabled successfully.

Now, we need to transfer our client cert to the APISIX container to verify the mTLS functionality.

```bash
# Transfer client certificate
kubectl -n apisix cp ./user.key <APISIX_POD_NAME>:/tmp/user.key
kubectl -n apisix cp ./user.pem <APISIX_POD_NAME>:/tmp/user.pem

# Test
kubectl -n apisix exec -it <APISIX_POD_NAME> -- curl --resolve 'mtls.httpbin.local:9443:127.0.0.1' "https://mtls.httpbin.local:9443/ip" -k --cert /tmp/user.pem --key /tmp/user.key
```

Parameter `--cert` and `--key` indicates our certificate and key path.

It should output normally:

```json
{
  "origin": "127.0.0.1"
}
```
