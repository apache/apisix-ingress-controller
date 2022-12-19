---
title: Match Stream Route with SNI
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

SNI(Server Name Indication) is an extension to the TLS protocol which allows a client to indicate which hostname it is attempting to connect to at the start of the TCP handshaking process.
Instead of requiring a different IP address for each SSL site, you can use SNI to install and configure multiple SSL sites to one IP address.

This guide walks through how to use the ApisixTls and ApisixRoute to route TLS-encrypted traffic to the TCP-based services with SNI.

## Prerequisites

- an available Kubernetes cluster.
- an available APISIX and APISIX Ingress Controller installation.
 
First of all, when installing APISIX, we need to enable TLS for the TCP address for APISIX in the Helm Chart, assume that enable tls on TCP port 6379.

```yaml
gateway:
  # L4 proxy (TCP/UDP)
  stream:
    enabled: true
    tcp:
      - addr: 6379
        tls: true
    udp: []
```

## Deploy Redis service

Before configuring the APISIX, we need to create 2 Redis services for testing.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: redis-1
  name: redis-1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis-1
  template:
    metadata:
      labels:
        app: redis-1
    spec:
      containers:
      - image: redis:6.2.7
        name: redis
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: redis-1
  name: redis-1
spec:
  ports:
  - port: 6379
    protocol: TCP
    targetPort: 6379
  selector:
    app: redis-1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: redis-2
  name: redis-2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis-2
  template:
    metadata:
      labels:
        app: redis-2
    spec:
      containers:
      - image: redis:6.2.7
        name: redis
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: redis-2
  name: redis-2
spec:
  ports:
  - port: 6379
    protocol: TCP
    targetPort: 6379
  selector:
    app: redis-2
```

## Create the certificates

When using SNI with TLS, a valid certificate is required for each domain or hostname that you want to use with SNI. 
This is because SNI allows multiple hostnames to be served from the same IP address and port, and the certificate is used to verify the identity of the server and establish an encrypted connection with the client.

Use OpenSSL to generate the certificate file and the key file for 2 Redis services.

```bash
openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 -keyout redis-1.key -out redis-1.crt -subj "/CN=redis-1.com"
openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 -keyout redis-2.key -out redis-2.crt -subj "/CN=redis-2.com"
```

Use kubectl with the tls secret type to create the Secrets using the certificate file and the key file.

```bash
kubectl create secret tls redis-1-secret --cert=./redis-1.crt --key=./redis-1.key
kubectl create secret tls redis-2-secret --cert=./redis-2.crt --key=./redis-2.key
```

## Create ApisixTls associated with Secret

Create ApisixTls associated with Secret resource, note the hosts field should match the domain or hostname in the certificate.
The apisix-ingress-controller will generate an APISIX SSL object in the APISIX.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: redis-1-tls
spec:
  hosts:
    - redis-1.com
  secret:
    name: redis-1-secret
    namespace: default
---
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: redis-2-tls
spec:
  hosts:
    - redis-2.com
  secret:
    name: redis-2-secret
    namespace: default
```
## Create ApisixRoute match the stream route with SNI

Define the route for proxying two Redis services traffic through APISIX. Specify the `spec.stream.match.host` field to match the stream route with SNI.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: redis-stream-route
spec:
  stream:
  - name: redis-1
    protocol: TCP
    match:
      ingressPort: 6379
      host: redis-1.com 
    backend:
      serviceName: redis-1
      servicePort: 6379
  - name: redis-2
    protocol: TCP
    match:
      ingressPort: 6379
      host: redis-2.com
    backend:
      serviceName: redis-2
      servicePort: 6379
```

## Test

Let's verify the configuration. In order to access APISIX locally, we can use `kubectl port-forward` command to forward traffic from the specified port at your local machine to the specified port on the specified service.

```bash
kubectl port-forward -n ingress-apisix svc/apisix-gateway 6379:6379
```

Now, connect to 2 Redis services, and set a key named `server`, with different values to distinguish 2 Redis services. 

```bash
# connect to the redis-1 server
redis-cli -h 127.0.0.1 -p 6379 --tls --sni redis-1.com --insecure
127.0.0.1:6379> set server redis-1
OK

# connect to the redis-2 server
redis-cli -h 127.0.0.1 -p 6379 --tls --sni redis-2.com --insecure
127.0.0.1:6379> set server redis-2
OK
```

Then connect to Redis services again to check whether routing based on SNI is successful.

```bash
redis-cli -h 127.0.0.1 -p 6379 --tls --sni redis-1.com --insecure
127.0.0.1:6379> get server
"redis-1"

redis-cli -h 127.0.0.1 -p 6379 --tls --sni redis-2.com --insecure
127.0.0.1:6379> get server
"redis-2"
```