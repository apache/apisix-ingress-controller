---
title: 用 Ingress 资源代理 httpbin 服务示例
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

本文档介绍了 apisix-ingress-controller 如何通过 [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) 引导 Apache APISIX 将流量正确路由到 httpbin 服务上。

## 前提条件

* 工作环境准备可用的 Kubernetes 集权, 建议使用 [Minikube](https://github.com/kubernetes/minikube)
* [在 Kubernetes 上用 Helm Chart 安装 Apache APISIX](https://github.com/apache/apisix-helm-chart)
* 安装 [apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller/blob/master/docs/install.md)

## 部署 httpbin 服务

我们使用了 [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/) 这个服务镜像, 有关详情请参见其概述页.

现在, 在 Kubernetes 集群中部署该服务:

```shell
kubectl run httpbin --image kennethreitz/httpbin --port 80
kubectl expose pod httpbin --port 80
```

## 资源交付

先创建一个 Ingress 资源:

```yaml
# httpbin-ingress.yaml
# Note use apiVersion is networking.k8s.io/v1, so please make sure your
# Kubernetes cluster version is v1.19.0 or higher.
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpserver-ingress
spec:
  # apisix-ingress-controller is only interested in Ingress
  # resources with the matched ingressClass name, in our case,
  # it's apisix.
  ingressClassName: apisix
  rules:
  - host: local.httpbin.org
    http:
      paths:
      - backend:
          service:
            name: httpbin
            port:
              number: 80
        path: /
        pathType: Prefix

# Use ingress.networking.k8s.io/v1beta1 if your Kubernetes cluster
# version is older than v1.19.0.
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: httpserver-ingress
  # Note for ingress.networking.k8s.io/v1beta1,
  # you have to carry annotation kubernetes.io/ingress.class,
  # and its value must be matched with the one configured in
  # apisix-ingress-controller, in our case, it's apisix.
  annotations:
    kubernetes.io/ingress.class: apisix
spec:
  rules:
    - host: local.httpbin.org
      http:
        paths:
          - backend:
              serviceName: httpbin
              servicePort: 80
            path: /
            pathType: Prefix
```

上面这个 YAML 代码片段显示了一个简单的 Ingress 配置，该配置告诉 Apache APISIX 所有 header 参数中带有 Host 为 `local.httpbin.org` 的请求都会被路由到 `httpbin` 服务。

现在来创建服务。

```shell
kubectl apply -f httpbin-ingress.yaml
```

## 测试

在 Apache APISIX Pod 中执行 curl 调用来检查资源是否已发送到它。

注意：在 Apache APISIX 集群中，应该将 `--apisix-admin-key` 的值替换为真正的 `admin_key` 的值。如果你的工作环境是windows环境，使用如下命令请注意替换 `'` 为 `"`, 否则可能会使curl命令执行不完整，影响测试的结果（会因为 X-API-Key 参数带不到服务器上，返回 `401` 的响应码）

```shell
kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9180/apisix/admin/routes -H 'X-API-Key: edd1c9f034335f136f87ad84b625c8f1'
```

请求 Apache APISIX 取校验路由。

```shell
kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9080/headers -H 'Host: local.httpbin.org'
```

如果成功, 你会看到如下的 JSON 串，包含了所有的请求 headers :

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
