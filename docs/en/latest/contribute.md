---
title: Developing for Apache APISIX Ingress Controller
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

This document explains how to get started with developing for Apache APISIX Ingress controller.

## Prerequisites

* Install [Go 1.13](https://golang.org/dl/) or later, and we use go module to manage the go package dependencies.
* Prepare an available Kubernetes cluster in your workstation, we recommend you to use [KIND](https://kind.sigs.k8s.io/).
* Install Apache APISIX in Kubernetes by [Helm Chart](https://github.com/apache/apisix-helm-chart).

## Fork and Clone

* Fork the repository from [apache/apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller) to your own GitHub account.
* Clone the fork repository to your workstation.
* Run `go mod download` to download modules to local cache. By the way, if you are a developer in China, we suggest you setting `GOPROXY` to `https://goproxy.cn` to speed up the downloading.

## Build

```shell
cd /path/to/apisix-ingress-controller
make build
./apisix-ingress-controller version
```

## How to add a new feature or change an existing one

Before making any significant changes, please [open an issue](https://github.com/apache/apisix-ingress-controller/issues). Discussing your proposed changes ahead of time will make the contribution process smooth for everyone.

Once we've discussed your changes and you've got your code ready, make sure that tests are passing and open your pull request. Your PR is most likely to be accepted if it:

* Update the README.md with details of changes to the interface.
* Includes tests for new functionality.
* References the original issue in the description, e.g. "Resolves #123".
* Has a [good commit message](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html).

## Test

### Run unit test cases

```shell
cd /path/to/apisix-ingress-controller
make unit-test
```

### Run e2e test cases

We using [KIND](https://kind.sigs.k8s.io/) for running e2e test cases. Please ensure `kind` CLI has been installed.
Currently we using KIND latest version v0.11.1 and using Kubernetes v1.21.1 for testing.

```shell
cd /path/to/apisix-ingress-controller
make e2e-test-local
```

Note the running of e2e cases is somewhat slow, so please be patient.

> See [here](https://onsi.github.io/ginkgo/#focused-specs) to learn
how to just run partial e2e cases.

### Build docker image

Suppose our image tag is `a.b.c`:

```shell
cd /path/to/apisix-ingress-controller
make build-image IMAGE_TAG=a.b.c
```

> Note: The Dockerfile in this repository is only for development, not for release.

If you're coding for apisix-ingress-controller and adding some e2e test cases to verify your changes,
you should push the images to the image registry that your Kubernetes cluster can access, if you're using Kind, just run the following command:

```shell
make push-images IMAGE_TAG=a.b.c
```

## Run apisix-ingress-controller locally

We assume all prerequisites above mentioned are met, what's more, since we want to run apisix-ingress-controller in bare-metal environment, please make sure both the proxy service and admin api service of Apache APISIX are exposed outside of the Kubernetes cluster, e.g. configuring them as [NodePort](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport) services.

Also, we can also use `port-forward` to expose the Admin API port of Apache APISIX Pod. The default port of Apache APISIX Admin API is 9180, next I'll expose the local port `127.0.0.1:9180`:

```shell
kubectl port-forward -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} 9180:9180
```

Run apisix-ingress-controller:

```shell
cd /path/to/apisix-ingress-controller
./apisix-ingress-controller ingress \
    --kubeconfig /path/to/kubeconfig \
    --http-listen :8080 \
    --log-output stderr \
    --default-apisix-cluster-base-url http://127.0.0.1:9180/apisix/admin \
    --default-apisix-cluster-admin-key edd1c9f034335f136f87ad84b625c8f1
```

Something you need to pay attention to:

* configuring of `--kubeconfig`, if you are using Minikube, the file path should be `~/.kube/config`.
* configuring of `--default-apisix-cluster-admin-key`, if you have changed the admin key in Apache APISIX, also changing it here. If you have disabled the authentication in Apache APISIX, just removing this option.

## Pre-commit todo

When everything is ready, before submitting the code, please make sure that the license, code style, and document format are consistent with the project specification.

We provide commands to implement it, just run the following commands:

```shell
make update-codegen
make update-license
make update-gofmt
make update-mdlint
```

or just run one command:

```shell
make update-all
```
