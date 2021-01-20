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

# Install Ingress APISIX on K3S

This document explains how to install Ingress APISIX on [k3S](https://k3s.io/).

K3S is a certified Kubernetes distribution built for IoT and Edge computing, whilst [Apache APISIX](https://apisix.apache.org) is also good at IoT (See [MQTT plugin](https://github.com/apache/apisix/blob/master/doc/plugins/mqtt-proxy.md)) and runs well on ARM architecture.
It's a good choice to use Ingress APISIX as the north-south API gateway in K3S.

## Prerequisites

* Install [K3S](https://rancher.com/docs/k3s/latest/en/installation/).
* Install [Helm](https://helm.sh/).
* Clone [Apache APISIX Charts](https://github.com/apache/apisix-helm-chart).
* Clone [apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller).
* Make sure your target namespace exists, kubectl operations thorough this document will be executed in namespace `ingress-apisix`.

## Install APISIX

[Apache APISIX](http://apisix.apache.org/) as the proxy plane of apisix-ingress-controller, should be deployed in advance.

```shell
cd /path/to/apisix-helm-chart
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update ./chart/apisix
helm install apisix ./chart/apisix \
  --set gateway.type=NodePort \
  --set allow.ipList="{0.0.0.0/0}" \
  --namespace ingress-apisix \
  --kubeconfig /etc/rancher/k3s/k3s.yaml
kubectl get service --namespace ingress-apisix
```

*Note root permission may required to run above commands.*

Two Service resources were created, one is `apisix-gateway`, which processes the real traffic; another is `apisix-admin`, which acts as the control plane to process all the configuration changes.

The gateway service type is set to `NodePort`, so that clients can access Apache APISIX through the Node IPs and the assigned port. If you want to expose a `LoadBalancer` service, try to use [Klipper](https://github.com/k3s-io/klipper-lb).

Another thing should be concerned that the `allow.ipList` field should be customized according to the [K3S Networking Settings](https://rancher.com/docs/k3s/latest/en/installation/install-options/server-config/#networking), so that the apisix-ingress-controller instances can access the APISIX instances (resources pushing).

## Install apisix-ingress-controller

You can also install apisix-ingress-controller by Helm Charts, it's recommended to install it in the same namespace with Apache APISIX.

```shell
cd /path/to/apisix-ingress-controller
# install base resources, e.g. ServiceAccount.
helm install ingress-apisix-base -n ingress-apisix ./charts/base --kubeconfig /etc/rancher/k3s/k3s.yaml
# install apisix-ingress-controller
helm install ingress-apisix ./charts/ingress-apisix \
  --set ingressController.image.tag=dev \
  --set ingressController.config.apisix.baseURL=http://apisix-admin:9180/apisix/admin \
  --set ingressController.config.apisix.adminKey=edd1c9f034335f136f87ad84b625c8f1 \
  --namespace ingress-apisix \
  --kubeconfig /etc/rancher/k3s/k3s.yaml
```

*Note root permission may required to run above commands.*

The admin key used in abovementioned commands is the default one, if you change the admin key configuration when you deployed APISIX, please remember to change it here.

Change the `ingressController.image.tag` to the Apache APISIX version that you desire. You have to wait for while until the correspdoning pods are running.

Now try to create some [resources](../CRD-specification.md) to verify the running status. As a minimalist example, see [proxy-the-httpbin-service](../samples/proxy-the-httpbin-service.md) to learn how to apply resources to drive the apisix-ingress-controller.
