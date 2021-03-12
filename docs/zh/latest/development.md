---
title: Apache APISIX Ingress Controller 开发指南
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

本文档主要阐述如何进行 Apache APISIX Ingress controller 的开发工作.

## 前提条件

* 安装 [Go 1.13](https://golang.org/dl/) +，我们使用go模块能进行管理包的依赖。如果在国内，建议访问此 [url](https://golang.google.cn/)
* 准备一个可用的 Kubernetes cluster 环境, 推荐使用 [Minikube](https://github.com/kubernetes/minikube)。
* [在 Kubernetes 环境中用 Helm Chart 方式安装 Apache APISIX](https://github.com/apache/apisix-helm-chart)。

## Fork 和 Clone

* Fork 项目 [apache/apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller) 到自己的 GitHub 账号。
* Clone 项目到自己的工作环境。
* 运行 `go mod download` 下载模块到本地缓存。 顺便提一下, 如果你是国内的开发者, 建议你设置代理 `GOPROXY` 到 `https://goproxy.cn` 方便下载加速。

## 构建

```shell
cd /path/to/apisix-ingress-controller
make build
./apisix-ingress-controller version
```

## 测试

### 运行单元测试用例

```shell
cd /path/to/apisix-ingress-controller
make unit-test
```

### 运行 e2e 测试用例

```shell
cd /path/to/apisix-ingress-controller
make e2e-test
```

注意 e2e 测试用例的执行有时候比较慢， 请保持耐心。

### 构建 docker 镜像

```shell
cd /path/to/apisix-ingress-controller
make build-image IMAGE_TAG=a.b.c
```

注意项目中的Dockerfile 是为开发准备的， 不能用来发版。

## 本地运行 apisix-ingress-controller

假设上述所有前提条件都满足，而且，我们希望在裸机环境下运行 apisix-ingress-controller ，请确保 ApacheApiSix 的 proxy 服务和 admin api 服务都暴露在 Kubernetes 集群之外，可以将它们配置为 [NodePort](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport)

假设 Admin API service 的地址是 `http://192.168.65.2:31156`. 那么 ingress-apisix-controller 下一步需要执行的命令是：

```shell
cd /path/to/apisix-ingress-controller
./apisix-ingress-controller ingress \
    --kubeconfig /path/to/kubeconfig \
    --http-listen :8080 \
    --log-output stderr \
    --apisix-base-url http://192.168.65.2:31156/apisix/admin
    --apisix-admin-key edd1c9f034335f136f87ad84b625c8f1
```

主要注意的几点:

* `--kubeconfig` 配置项, 如果使用的是 Minikube 环境， 配置的目录路径应该是 `~/.kube/config`。
* `--apisix-admin-key` 配置项, 如果在 Apache APISIX 中更改了管理密钥，这里也需要同步更改，如果您禁用了 Apache APISIX 的身份验证，只需删除此选项即可。
