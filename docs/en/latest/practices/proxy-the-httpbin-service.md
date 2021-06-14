---
title: Proxy the httpbin service
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

This document explains how apisix-ingress-controller guides Apache APISIX routes traffic to httpbin service correctly.

## Prerequisites

* Prepare an available Kubernetes cluster in your workstation, we recommend you to use [Minikube](https://github.com/kubernetes/minikube).
* Install Apache APISIX in Kubernetes by [Helm Chart](https://github.com/apache/apisix-helm-chart).
* Install [apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller/blob/master/install.md).

## Deploy httpbin service

We use [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/) as the service image, See its overview page for details.

Now, try to deploy it to your Kubernetes cluster:

```shell
kubectl run httpbin --image kennethreitz/httpbin --port 80
kubectl expose pod httpbin --port 80
```

## Resource Delivery

In order to let Apache APISIX proxies requests to httpbin, we need to create an `ApisixRoute` resource, if you're not familiar with it, see the [reference](https://github.com/apache/apisix-ingress-controller/blob/master/samples/deploy/crd/v1beta1/ApisixRoute.yaml) for the details.

```yaml
# httpbin-route.yaml
apiVersion: apisix.apache.org/v2alpha1
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
    backend:
        serviceName: httpbin
        servicePort: 80
```

The YAML snippet shows a simple `ApisixRoute` configuration, which tells Apache APISIX to route all requests with Host `local.httpbin.org` to the `httpbin` service.
Now try to create it.

```shell
kubectl apply -f httpbin-route.yaml
```

## Test

Run curl call in one of Apache APISIX Pods to check whether the resource was delivered to it. Note you should replace the value of `--apisix-admin-key` to the real `admin_key` value in your Apache APISIX cluster.

```shell
kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9180/apisix/admin/routes -H 'X-API-Key: edd1c9f034335f136f87ad84b625c8f1'
```

And request to Apache APISIX to verify the route.

```shell
kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9080/headers -H 'Host: local.httpbin.org'
```

In case of success, you'll see a JSON string which contains all requests headers carried by `curl` like:

```json
{
  "headers": {
    "Accept": "*/*",
    "Host": "httpbin.org",
    "User-Agent": "curl/7.64.1",
    "X-Amzn-Trace-Id": "Root=1-5ffc3273-2928e0844e19c9810d1bbd8a"
  }
}
```
