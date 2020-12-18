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

## Dependencies

* Kubernetes
* [Deploy Apache APISIX in k8s](https://github.com/apache/apisix/blob/master/kubernetes/README.md)

To install `ingress controller` in k8s, need to care about 3 parts:

1. CRDs: The definitions of Apache APISIX configurations in Kubernetes.

2. [RBAC](https://kubernetes.io/blog/2017/04/rbac-support-in-kubernetes/): This is support by Kubernetes, granting `ingress controller` resource access permissions.

3. Configmap: Contains the necessary configuration for `ingress controller`.

## Kustomize

Before executing the following command, you need to create the namespace `ingress-apisix`:

```shell
kubectl create ns ingress-apisix
```

Install the abovementioned resources by [Kustomize](https://kustomize.io/):

```shell
kubectl kustomize "github.com/apache/apisix-ingress-controller/samples/deploy?ref=master" | kubectl apply -f -
```

If the default parameters in samples/deploy are not good for you, just tweak them and run:

```shell
kubectl apply -k /path/to/apisix-ingress-controller/samples/deploy
```

## Helm

// todo

## Master branch builds

```shell
docker build -t apache/ingress-controller:v0.1.0 ../.
```

## Next

* [Usage](./usage.md)
