---
title: Development
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - Development
  - Contribute
description: Setting up development environment for APISIX Ingress controller.
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

This document walks through how you can set up your development environment to contribute to APISIX Ingress controller.

## Prerequisites

Before you get started make sure you have:

1. Installed [Go 1.13](https://golang.org/dl/) or later
2. A Kubernetes cluster available. We recommend using [kind](https://kind.sigs.k8s.io/).
3. Installed APISIX in Kubernetes using [Helm](https://github.com/apache/apisix-helm-chart).

## Fork and clone

1. Fork the repository [apache/apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller) to your GitHub account
2. Clone the fork to your workstation.
3. Run `go mod download` to download the required modules.

:::tip

If you are in China, you can speed up the downloads by setting `GOPROXY` to `https://goproxy.cn`.

:::

## Build from source

To build APISIX Ingress controller, run the command below on the root of the project:

```shell
make build
```

Now you can run it by:

```shell
./apisix-ingress-controller version
```

## Making changes

Prior to opening a pull request with changes or new features, please [open an issue](https://github.com/apache/apisix-ingress-controller/issues).

Make sure that the license, code style, and document format are consistent with the project specification. You can do this by running:

```shell
make update-all
```

Your pull requests will more likely to be accepted if:

1. All tests are passing and tests are included for new functionalities
2. README and documentation is updated with the chages
3. PR is linked to the issue with semantic keywords ("fixes #145")
4. Has detailed PR descriptions and good [commit messages](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)

## Running tests

To run unit tests:

```shell
make unit-test
```

To run end-to-end tests, you need to install [kind](https://kind.sigs.k8s.io/).

Currently, we use Kind version `0.11.1` and Kubernetes version `1.21.1` for running the tests.

```shell
make e2e-test-local
```

:::note

End-to-end tests are comprehensive and takes time to run.

See [focused specs](https://onsi.github.io/ginkgo/#focused-specs) to learn how you can run partial tests.

:::

To run e2e tests on any changes, you can build a Docker image and push it to a registry that the Kubernetes cluster can access:

```shell
make build-image IMAGE_TAG=a.b.c
make push-images IMAGE_TAG=a.b.c
```

## Running locally

Make sure to expose both the APISIX proxy and the Admin API to outside the Kubernetes cluster. You can use [NodePort service](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport) or use `port-forward` as shown below:

Also, we can also use `port-forward` to expose the Admin API port of Apache APISIX Pod. The default port of Apache APISIX Admin API is 9180, next I'll expose the local port `127.0.0.1:9180`:

```shell
kubectl port-forward -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} 9180:9180
```

This will expose the default `9180` port of the Admin API to outside the cluster.

You can now run APISIX Ingress controller by:

```shell
cd /path/to/apisix-ingress-controller
./apisix-ingress-controller ingress \
    --kubeconfig /path/to/kubeconfig \
    --http-listen :8080 \
    --log-output stderr \
    --default-apisix-cluster-base-url http://127.0.0.1:9180/apisix/admin \
    --default-apisix-cluster-admin-key edd1c9f034335f136f87ad84b625c8f1
```

:::note

1. If you are using minikube, the path to kubeconfig should be `~/.kube/config`.
2. If you have changed the default Admin API key, you have to pass it in the `--default-apisix-cluster-admin-key` flag. You can remove the flag if you have disabled authentication.

:::
