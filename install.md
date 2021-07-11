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

* [Install Ingress APISIX on Azure AKS](./docs/en/latest/deployments/azure.md)
* [Install Ingress APISIX on AWS EKS](./docs/en/latest/deployments/aws.md)
* [Install Ingress APISIX on ACK](./docs/en/latest/deployments/ack.md)
* [Install Ingress APISIX on Google Cloud GKE](./docs/en/latest/deployments/gke.md)
* [Install Ingress APISIX on Minikube](./docs/en/latest/deployments/minikube.md)
* [Install Ingress APISIX on KubeSphere](./docs/en/latest/deployments/kubesphere.md)
* [Install Ingress APISIX on K3S and RKE](./docs/en/latest/deployments/k3s-rke.md)

## Kustomize Support

As an alternative way, you can also choose to install apisix-ingress-controller by [Kustomize](https://kustomize.io/).

```shell
kubectl create namespace ingress-apisix
kubectl kustomize "github.com/apache/apisix-ingress-controller/samples/deploy?ref=master" | kubectl apply -f -
```

Parameters are hardcoded so if the default values are not good for you, just tweak them manually.

To tweak parameters, first you need to modify config files in _samples/deploy_ directory. There are many ways to acheive this. For example, you may insert a `sed` command after `kubectl kustomize`, that is, `kubectl kustomize "github.com/apache/apisix-ingress-controller/samples/deploy?ref=master" | sed "s@to-be-modified@after-modified@g" | kubectl apply -f -`. Another way is to use a local copy of _samples/deploy_ directory or a copy from your repo.

Then you need to know which parameter need to be tweaked. If APISIX access token or the address of APISIX Admin API is changed, you need to modify `apisix.admin_key` or `apisix.base_url` respectively in field `.data.config.yaml` in file _samples/deploy/configmap/apisix-ingress-cm.yaml_. Another example is to install apisix-ingress-controller with different version, in which case you need to configure `.spec.template.spec.containers.[image]` to a desired version in file _samples/deploy/deployment/ingress-controller.yaml_.

## Verify Installation

There are a lot of use examples (See [samples](docs/en/latest/practices/index.md) for more details), try to follow the operations there to verify the installation.
