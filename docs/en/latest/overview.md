---
title: Overview
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
description: Overview of the APISIX Ingress Controller, its features, APISIX compatibility, and how to contribute to the project.
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

APISIX Ingress Controller is a [Kubernetes ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) using [Apache APISIX](https://apisix.apache.org) as the high performance reverse proxy.

APISIX Ingress Controller can be configured using the native Kubernetes Ingress or Gateway API, as well as with APISIX’s own declarative and easy-to-use custom resources. The controller translates these resources into APISIX configuration.

See the [Getting Started tutorials](./getting-started/get-apisix-ingress-controller.md) to set up and start using the APISIX Ingress Controller.

## Features

To summarize, APISIX ingress controller has the following features:

- Declarative configuration with CRDs.
- Supports native Kubernetes Ingress v1 and Gateway API.
- Supports service discovery through Kubernetes Service.
- Supports load balancing based on pods (Upstream nodes).
- Rich [Plugins](https://apisix.apache.org/docs/apisix/next/plugins/batch-requests/) with [custom Plugin](https://apisix.apache.org/docs/apisix/next/plugin-develop/) support.

## Get involved

You can contribute to the development of APISIX ingress controller. See [Development guide](./developer-guide.md) for instructions on setting up the project locally.

See the [Contribute to APISIX](https://apisix.apache.org/docs/general/contributor-guide/) section for details on the contributing flow.

## Compatibility with APISIX

The table below shows the compatibility between APISIX ingress controller and the APISIX proxy.

:::note

APISIX Ingress Controller 2.0.0+ support the [APISIX Standalone API-driven Mode](https://apisix.apache.org/docs/apisix/deployment-modes/#api-driven-experimental), but require APISIX 3.13+.

:::

| APISIX ingress controller | Supported APISIX versions | Recommended APISIX version |
| ------------------------- | ------------------------- | -------------------------- |
| `master`                  | `>=3.0`                   | `3.13`                     |
| `2.0.0`                   | `>=3.0`                   | `3.13`                     |
| `1.6.0`                   | `>= 2.15`, `>=3.0`        | `2.15`, `3.0`              |
| `1.5.0`                   | `>= 2.7`                  | `2.15`                     |
| `1.4.0`                   | `>= 2.7`                  | `2.11`                     |
| `1.3.0`                   | `>= 2.7`                  | `2.10`                     |
| `1.2.0`                   | `>= 2.7`                  | `2.8`                      |
| `1.1.0`                   | `>= 2.7`                  | `2.7`                      |
| `1.1.0`                   | `>= 2.7`                  | `2.7`                      |
| `1.0.0`                   | `>= 2.7`                  | `2.7`                      |
| `0.6`                     | `>= 2.6`                  | `2.6`                      |
| `0.5`                     | `>= 2.4`                  | `2.5`                      |
| `0.4`                     | `>= 2.4`                  |                            |
