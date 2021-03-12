---
title: 在 K3S and Rancher RKE 上安装 Ingress APISIX 
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

本文档介绍如何在 [k3S](https://k3s.io/) and [Rancher RKE](https://rancher.com/products/rke/)上安装 Ingress APISIX。

K3S 是一个专为物联网和边缘计算服务的经过认证的 Kubernetes 发行版，而 [Apache APISIX](https://apisix.apache.org) 也擅长物联网（参见[MQTT插件](https://github.com/apache/apisix/blob/master/doc/plugins/mqtt-proxy.md)) 在ARM架构上运行良好。

在 K3S 中使用入口 APISIX 作为南北向 API 网关是一个不错的选择。

## 前提条件

* 安装 [K3S](https://rancher.com/docs/k3s/latest/en/installation/) 或者 [Rancher RKE](https://rancher.com/docs/rke/latest/en/installation/)
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
  --kubeconfig /etc/rancher/k3s/k3s.yaml
kubectl get service --namespace ingress-apisix
```

*如果你正在使用 K3S, 默认的 kubeconfig 文件是在 /etc/rancher/k3s 里，因此 root 执行权限可能是需要的.*

两个 Service 资源会被创建，一个是 `apisix-gateway`，处理实际流量；另一个是`apisix-admin`，它充当控制面板来处理所有配置更改。

服务网关类型要设置成 `NodePort`, 这样客户机就可以通过节点的IPs 和分配到端口来访问 Apache APISIX 。如果你正在使用 K3S 并想公开一个`LoadBalancer` 服务, 尝试使用 [Klipper](https://github.com/k3s-io/klipper-lb)。

另外一个值得注意的是 `allow.ipList` 字段应根据 Pod CIDR 设置规则（ 详见 [K3S](https://rancher.com/docs/k3s/latest/en/installation/install-options/server-config/#networking) 或者 [Rancher RKE](https://rancher.com/docs/rancher/v2.x/en/cluster-provisioning/rke-clusters/options/#cluster-config-file) ）进行定制， 以便 apisix-ingress-controller 实例可以访问 APISIX 实例（ 资源推送 ）。

## 安装 apisix-ingress-controller

您还可以通过 Helm Charts 安装 apisix-ingress-controller，建议将其安装在与 Apache APISIX 相同的 namespace 中。

```shell
cd /path/to/apisix-helm-chart
# install apisix-ingress-controller
helm install apisix-ingress-controller ./charts/apisix-ingress-controller \
  --set image.tag=dev \
  --set config.apisix.baseURL=http://apisix-admin:9180/apisix/admin \
  --set config.apisix.adminKey=edd1c9f034335f136f87ad84b625c8f1 \
  --namespace ingress-apisix \
  --kubeconfig /etc/rancher/k3s/k3s.yaml
```

*如果你正在使用 K3S, 默认的 kubeconfig 文件是在 /etc/rancher/k3s 里，因此 root 执行权限可能是需要的.*

上面提到的命令中使用的管理密钥是默认的，如果您在部署 APISIX时更改了管理密钥配置，请记住在此处进行更改。

更改 `image.tag` 到您想要的 apisix-ingress-controller 版本。 您必须等待一段时间， 直到相应的 pod 运行。

现在尝试创建一些 [资源](../CRD-specification.md) 来验证 Ingress APISIX 的运行。 作为一个最简单的例子，请参见 [代理 httpbin 服务示例](../practices/proxy-the-httpbin-service.md) 来了解如何应用资源来驱动 apisix-ingress-controller 。
