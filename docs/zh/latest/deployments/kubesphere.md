---
title: 在 KubeSphere 上安装 Ingress APISIX 
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

本文档介绍如何在 [KubeSphere](https://kubesphere.io/)上安装 Ingress APISIX。

KubeSphere 是一个用 Kubernetes 为内核来管理云原生应用程序 的分布式操作系统，并为第三方应用程序的无缝集成提供即插即用的架构，以增强其生态。

## 准备工作

* 安装 [KubeSphere](https://kubesphere.io/docs/quick-start/), 可选 [All-in-one Installation on Linux](https://kubesphere.io/docs/quick-start/all-in-one-on-linux/) 或者 [Minimal KubeSphere on Kubernetes](https://kubesphere.io/docs/quick-start/minimal-kubesphere-on-k8s/)两种安装方法
* 安装 [Helm](https://helm.sh/)
* Clone [Apache APISIX Charts](https://github.com/apache/apisix-helm-chart)项目
* 确保目标命名空间存在，此文档中的 kubectl 操作将在命名空间 `ingress-apisix` 中执行

## 安装 APISIX

[Apache APISIX](http://apisix.apache.org/) 作为 apisix-ingress-controller 的代理层, 应该提前部署。

```shell
cd /path/to/apisix-helm-chart
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update ./charts/apisix
helm install apisix ./charts/apisix \
  --set gateway.type=NodePort \
  --set allow.ipList="{0.0.0.0/0}" \
  --namespace ingress-apisix \
kubectl get service --namespace ingress-apisix
```

两个 Service 资源会被创建，一个是 `apisix-gateway`，处理实际流量；另一个是`apisix-admin`，它充当控制面板来处理所有配置更改。

服务网关类型要设置成 `NodePort`, 这样客户机就可以通过节点的IPs 和分配到端口来访问 Apache APISIX 。如果你想公开一个`LoadBalancer` 服务, 尝试使用 [Porter](https://github.com/kubesphere/porter)。

另外一个值得注意的是 `allow.ipList` 字段应根据 Pod CIDR 设置规则进行定制， 以便 apisix-ingress-controller 实例可以访问 APISIX 实例（ 资源推送 ）。

## 安装 apisix-ingress-controller

您还可以通过 Helm Charts 安装 apisix-ingress-controller，建议将其安装在与 Apache APISIX 相同的 namespace 中。

```shell
cd /path/to/apisix-helm-chart
# install apisix-ingress-controller
helm install apisix-ingress-controller ./charts/apisix-ingress-controller \
  --set image.tag=dev \
  --set config.apisix.baseURL=http://apisix-admin:9180/apisix/admin \
  --set config.apisix.adminKey=edd1c9f034335f136f87ad84b625c8f1 \
  --namespace ingress-apisix
```

上面提到的命令中使用的管理密钥是默认的，如果您在部署 APISIX时更改了管理密钥配置，请记住在此处进行更改。

更改 `image.tag` 到您想要的 apisix-ingress-controller 版本。 您必须等待一段时间， 直到相应的 pod 运行。

现在尝试创建一些 [资源](../CRD-specification.md) 来验证 Ingress APISIX 的运行。 作为一个最简单的例子，请参见 [代理 httpbin 服务示例](../practices/proxy-the-httpbin-service.md) 来了解如何应用资源来驱动 apisix-ingress-controller 。
