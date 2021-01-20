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

# Installation

This is an index page about installing Ingress APISIX in several environments. Click the following links on demands.

* [Install Ingress APISIX on Minikube](deployments/minikube.md)
* [Install Ingress APISIX on K3S](deployments/k3s-rke.md)
* [Install Ingress APISIX on Azure AKS](deployments/azure.md)
* [Install Ingress APISIX on AWS EKS](deployments/aws.md)

## Kustomize Support

As an alternative way, you can also choose to install apisix-ingress-controller by [Kustomize](https://kustomize.io/).

```shell
kubectl create namespace ingress-apisix
kubectl kustomize "github.com/apache/apisix-ingress-controller/samples/deploy?ref=master" | kubectl apply -f -
```

Parameters are hardcoded so if the default values are not good for you, just tweak them manually.

## Verify Installation

There are a lot of use examples (See [samples](./samples/index.md) for more details), try to follow the operations there to verify the installation.
