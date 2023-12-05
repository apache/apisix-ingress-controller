---
title: Using APISIX Ingress as Istio Egress Gateway
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

Istio uses ingress and egress gateways to configure load balancers executing at the edge of a service mesh. An ingress gateway defines entry points into the mesh that all incoming traffic flows through. Egress gateway is a symmetrical concept; it defines exit points from the mesh.

Although APISIX Ingress was originally implemented as an Ingress Controller, it can still be used as Istio egress gateway. This article will describe how to use it.

## Environment Preparation

### Install Istio

Use the official `istioctl` provided by Istio to install Istio. Here we use the `default` profile of `istioctl`, which contains istio and ingress, but not egress, meets our needs.

```bash
istioctl install --set profile=default
```

### Install APISIX Ingress

In this article, we will run APISIX Ingress in the namespace `ingress-apisix`.

Note that we need to manually label this namespace as `istio-injection=disabled` to avoid APISIX Ingress being injected by Istio.

```bash
kubectl create ns ingress-apisix
kubectl label ns ingress-apisix istio-injection=disabled

helm install apisix apisix/apisix --create-namespace \
  --set service.type=NodePort \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix \
  --set apisix.ssl.enabled=true
```

### Create Test Workload

We will run our tests in the `test` namespace. Use the following command to create the required test pod.

```bash
kubectl create ns test
kubectl label ns test istio-injection=enabled

kubectl -n test run consumer --image curlimages/curl \
  --image-pull-policy IfNotPresent \
  --command -- sh -c "trap : TERM INT; sleep 99d & wait"
```

After the creation is complete, try sending a test request to make sure the service mesh is working properly.

```bash
kubectl -n test exec -it consumer -- curl httpbin.org/get
```

The request will return a result similar to the following.

```json
{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "httpbin.org",
    "User-Agent": "curl/7.87.0-DEV",
    "X-Amzn-Trace-Id": "....",
    "X-B3-Sampled": "0",
    "X-B3-Spanid": "....",
    "X-B3-Traceid": "....",
    "X-Envoy-Attempt-Count": "1",
    "X-Envoy-Peer-Metadata": "....",
    "X-Envoy-Peer-Metadata-Id": "sidecar~10.244.0.21~consumer.test~test.svc.cluster.local"
  },
  "origin": "222.0.222.222",
  "url": "http://httpbin.org/get"
}
```

This indicates that our traffic is being sent directly from the envoy sidecar.

## Egress Gateway for HTTP Traffic

### Configure Istio

Now we need to define some Istio resources to allow requests for `httpbin.org` to be sent to APISIX.

Define a `ServiceEntry` for `httpbin.org`：

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: httpbin
spec:
  hosts:
  - httpbin.org
  ports:
  - number: 80
    name: http-port
    protocol: HTTP
  - number: 443
    name: https
    protocol: HTTPS
  resolution: DNS
```

Create an egress `Gateway` resource for `httpbin.org`, port 80, and a `DestinationRule` for traffic directed to the egress gateway.

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: apisix-egress
spec:
  selector:
    app.kubernetes.io/name: apisix
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - httpbin.org
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: egressgateway-for-httpbin
spec:
  host: apisix-gateway.ingress-apisix.svc.cluster.local
  subsets:
  - name: httpbin
```

Note the `selector` of the `Gateway` resource here, which will select our APISIX Ingress. The `host` field in the `DestinationRule` is the FQDN of the APISIX Ingress service.

Define a `VirtualService` to direct traffic from the sidecars to the egress gateway and from the egress gateway to the external service:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: httpbin-via-egress-gateway
spec:
  hosts:
  - httpbin.org
  gateways:
  - apisix-egress
  - mesh
  http:
  - match:
    - gateways:
      - mesh
      port: 80
    route:
    - destination:
        host: apisix-gateway.ingress-apisix.svc.cluster.local
        subset: httpbin
        port:
          number: 80
      weight: 100
  - match:
    - gateways:
      - apisix-egress
      port: 80
    route:
    - destination:
        host: httpbin.org
        port:
          number: 80
      weight: 100
```

In the yaml, `mesh` is the internal gateway of Istio, and `apisix-egress` is the name of the `Gateway` resource we defined.

Now, our configuration in the Istio is complete. Try request again.

```bash
kubectl -n test exec -it consumer -- curl httpbin.org/get
```

It will return a APISIX error.

```json
{"error_msg":"404 Route Not Found"}
```

This error indicates that the traffic has been sent to APISIX correctly. However, APISIX does not handle the Istio resources. So we need to do some additional configuration to make APISIX Ingress generate the correct routing configuration for APISIX.

### Configure APISIX Ingress

In the Istio configuration, we have successfully directed traffic from the mesh to APISIX, but APISIX doesn't currently have any route.

Here, we will use the [external service](https://github.com/apache/apisix-ingress-controller/blob/master/docs/en/latest/tutorials/external-service.md) feature to direct traffic to the external service.

```yaml
apiVersion: apisix.apache.org/v2
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
    upstreams:
    - name: httpbin-upstream
---
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-upstream
spec:
  externalNodes:
  - type: Domain
    name: httpbin.org
```

This configuration defines the route to `httpbin.org` for APISIX. Now we should be able to access the service properly, send the request again to test it.

```bash
kubectl -n test exec -it consumer -- curl httpbin.org/get
```

The request will return something like this,

```json
{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "httpbin.org",
    "User-Agent": "curl/7.87.0-DEV",
    "X-Amzn-Trace-Id": "...",
    "X-B3-Sampled": "0",
    "X-B3-Spanid": "...",
    "X-B3-Traceid": "...",
    "X-Envoy-Attempt-Count": "1",
    "X-Envoy-Decorator-Operation": "apisix-gateway.ingress-apisix.svc.cluster.local:80/*",
    "X-Envoy-Peer-Metadata": "...",
    "X-Envoy-Peer-Metadata-Id": "sidecar~10.244.0.21~consumer.test~test.svc.cluster.local",
    "X-Forwarded-Host": "httpbin.org"
  },
  "origin": "10.244.0.21, 222.0.222.222",
  "url": "http://httpbin.org/get"
}
```

Notice that the `origin` field in the result will contain the pod ip of the test pod `consumer`, which indicates that our request was successfully sent to the target service via APISIX.
