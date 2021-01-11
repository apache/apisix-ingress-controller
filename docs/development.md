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

# Developing for Apache APISIX Ingress Controller

This document explains how to get started with developing for Apache APISIX Ingress controller.

## Prerequisites

* Install [Go 1.13](https://golang.org/dl/) or later, and we use go module to manage the go package dependencies.
* Prepare an available Kubernetes cluster in your workstation, we recommend you to use [Minikube](https://github.com/kubernetes/minikube).
* [Install Apache APISIX in Kubernetes by Helm Chart](https://github.com/apache/apisix-helm-chart).

## Fork and Clone

* Fork the repository from https://github.com/apache/apisix-ingress-controller to your own GitHub account.
* Clone the forked repository to your workstation.
* Run `go mod download` to download modules to local cache. By the way, if you are a developer in China, we suggest you setting `GOPROXY` to `https://goproxy.cn` to speed up the downloading.

## Build

```shell
$ cd /path/to/apisix-ingress-controller
$ make build
$ ./apisix-ingress-controller version
```

## Test

### Run unit test cases

```shell
$ cd /path/to/apisix-ingress-controller
$ make unit-test
```

### Run e2e test cases

```shell
$ cd /path/to/apisix-ingress-controller
$ make e2e-test
```

Note the running of e2e cases is somewhat slow, so please be patient.

### Build docker image

```
$ cd /path/to/apisix-ingress-controller
$ make build-image IMAGE_TAG=a.b.c
```

The Dockerfile in this repository is only for development, not for release.

## Run apisix-ingress-controller locally

We assume all prerequisites abovementioned are met, what's more, since we want to run apisix-ingress-controller in bare-metal environment, please make sure both the proxy service and admin api service of Apache APISIX are exposed outside of the Kubernetes cluster, e.g. configuring them as [NodePort](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport) services. 

Let's assume the Admin API service address of Apache APISIX is `http://192.168.65.2:31156`. Next launch the ingress-apisix-controller by the following command.

```shell
$ cd /path/to/apisix-ingress-controller
$ ./apisix-ingress-controller ingress \
  --kubeconfig /path/to/kubeconfig \
  --http-listen :8080 \
  --log-output stderr \
  --apisix-base-url http://192.168.65.2:31156/apisix/admin
  --apisix-admin-key edd1c9f034335f136f87ad84b625c8f1
```

Something you need to pay attention to:

* configuring of `--kubeconfig`, if you are using Minikube, the file path should be `~/.kube/config`.
* configuring of `--apisix-admin-key`, if you have changed the admin key in Apache APISIX, also changing it here, if you disable the authentication if Apache APISIX, just removing this option.
