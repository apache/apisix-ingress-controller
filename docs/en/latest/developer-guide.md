---
title: Developer Guide
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

1. Installed [Go 1.23](https://golang.org/dl/) or later
2. A Kubernetes cluster available. We recommend using [kind](https://kind.sigs.k8s.io/).
3. Installed APISIX in Kubernetes using [Helm](https://github.com/apache/apisix-helm-chart).
4. Installed [ADC v0.20.0+](https://github.com/api7/adc/releases)

## Fork and clone

1. Fork the repository [apache/apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller) to your GitHub account
2. Clone the fork to your workstation.
3. Run `go mod download` to download the required modules.

:::tip

If you are in China, you can speed up the downloads by setting `GOPROXY` to `https://goproxy.cn`.

:::

## Install CRD and Gateway API

To install the [CRD](./concepts/resources.md#apisix-ingress-controller-crds-api) and [Gateway API](https://gateway-api.sigs.k8s.io/), run the following commands:

```shell
make install
```

## Build from source

To build APISIX Ingress controller, run the command below on the root of the project:

```shell
make build
```

Now you can run it by:

```shell
# for ARM64 architecture, use the following command:
# ./bin/apisix-ingress-controller_arm64 version
./bin/apisix-ingress-controller_amd64 version
```

## Building Image

To build a Docker image for APISIX Ingress controller, you can use the following command:

```shell
make build-image IMG=apache/apisix-ingress-controller:dev
```

## Deploying the Controller

To deploy the controller to your Kubernetes cluster, you can use the following command:

```shell
make deploy IMG=apache/apisix-ingress-controller:dev
```

To undeploy the controller from the cluster:

```shell
make undeploy
```

## Running tests

### Unit Tests

To run unit tests:

```shell
make unit-test
```

### e2e Tests

To run end-to-end tests, you need to install [kind](https://kind.sigs.k8s.io/).

Launch a kind cluster with the following command:

```shell
make kind-up
```

To run end-to-end e2e-tests against any changes, you need to load the built Docker images into the Kubernetes cluster:

```shell
# build docker image for APISIX Ingress controller
make build-image
# load the image into kind cluster
make kind-load-images
```

Currently, we use Kind version `0.26.0` and Kubernetes version `1.26+` for running the tests.

```shell
make e2e-test
```
