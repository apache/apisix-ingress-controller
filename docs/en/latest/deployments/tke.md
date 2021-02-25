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

# Install Ingress APISIX on Tencent TKE

This document explains how to install Ingress APISIX on [Tencent TKE](https://cloud.tencent.com/product/tke).

## Prerequisites

* Create a TKE Service on Tencent Cloud and make sure the API Server is accessible from your workspace.
* Install [Helm](https://helm.sh/).
* Download the kube config for your TKE Console.
* Clone [Apache APISIX Charts](https://github.com/apache/apisix-helm-chart).
* Make sure your target namespace exists, kubectl operations thorough this document will be executed in namespace `ingress-apisix`.

## Install APISIX

[Apache APISIX](http://apisix.apache.org/) as the proxy plane of apisix-ingress-controller, should be deployed in advance.

```shell
cd /path/to/apisix-helm-chart
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update ./charts/apisix
helm install apisix ./charts/apisix \
  --set gateway.type=LoadBalancer \
  --set allow.ipList="{0.0.0.0/0}" \
  --set etcd.persistence.size=10Gi \
  --namespace ingress-apisix \
kubectl get service --namespace ingress-apisix
```

Please be careful you must configure the `etcd.persistence.size` to multiplese of 10Gi (it's a limitation on TKE), otherwise the [PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) creation will fail.

Two Service resources were created, one is `apisix-gateway`, which processes the real traffic; another is `apisix-admin`, which acts as the control plane to process all the configuration changes.

The gateway service type is set to `LoadBalancer` (see [TKE Service Management](https://cloud.tencent.com/document/product/457/45487?from=10680) for more details), so that clients can access Apache APISIX through a load balancer. You can find the load balancer ip by running:

```shell
kubectl get service apisix-gateway --namespace ingress-apisix -o jsonpath='{.status.loadBalancer.ingress[].ip}'
```

Another thing should be concerned that the `allow.ipList` field should be customized according to the [TKE Network Settings](https://cloud.tencent.com/document/product/457/50353), so that the apisix-ingress-controller instances can access the APISIX instances (resources pushing).

## Install apisix-ingress-controller

You can also install apisix-ingress-controller by Helm Charts, it's recommended to install it in the same namespace with Apache APISIX.

```shell
cd /path/to/apisix-helm-chart
# install apisix-ingress-controller
helm install apisix-ingress-controller ./charts/apisix-ingress-controller \
  --set image.tag=dev \
  --set config.apisix.baseURL=http://apisix-admin:9180/apisix/admin \
  --set config.apisix.adminKey=edd1c9f034335f136f87ad84b625c8f1 \
  --namespace ingress-apisix
```

Change the `image.tag` to the apisix-ingress-controller version that you desire. You have to wait for while until the correspdoning pods are running.

Now open your [TKE console](https://console.cloud.tencent.com/tke2/overview), choosing your cluster and clicking the Workloads tag, you'll see all pods of Apache APISIX, etcd and apisix-ingress-controller are ready.

Try to create some [resources](../CRD-specification.md) to verify the running status. As a minimalist example, see [proxy-the-httpbin-service](../samples/proxy-the-httpbin-service.md) to learn how to apply resources to drive the apisix-ingress-controller.
