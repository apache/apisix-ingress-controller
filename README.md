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

# Apache APISIX for Kubernetes

[![Go Report Card](https://goreportcard.com/badge/github.com/apache/apisix-ingress-controller)](https://goreportcard.com/report/github.com/apache/apisix-ingress-controller)

Use [Apache APISIX](https://github.com/apache/apisix#apache-apisix) for Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

All configurations in `apisix-ingress-controller` are defined with Kubernetes CRDs (Custom Resource Definitions).
Support configuring [plugins](https://github.com/apache/apisix/blob/master/docs/en/latest/plugins), service registration discovery mechanism for upstreams, load balancing and more in Apache APISIX.

`apisix-ingress-controller` is an Apache APISIX control plane component. Currently it serves for Kubernetes clusters. In the future, we plan to separate the submodule to adapt to more deployment modes, such as virtual machine clusters.

The technical architecture of `apisix-ingress-controller`:

<img src="./docs/assets/images/module-0.png" alt="Architecture" width="743" height="559" />

## Status

This project is currently general availability.

## Features

* Declarative configuration for Apache APISIX with Custom Resource Definitions(CRDs), using k8s yaml struct with minimum learning curve.
* Hot-reload during yaml apply.
* Native Kubernetes Ingress (both `v1` and `v1beta1`) support.
* Auto register k8s endpoint to upstream (Apache APISIX) node.
* Support load balancing based on pod (upstream nodes).
* Out of box support for node health check.
* Plug-in extension supports hot configuration and immediate effect.
* Support SSL and mTLS for routes.
* Support traffic split and canary deployments.
* Support TCP 4 layer proxy.
* Ingress controller itself as a pluggable hot-reload component.
* Multi-cluster configuration distribution.

[More about comparison among multiple Ingress Controllers.](https://docs.google.com/spreadsheets/d/191WWNpjJ2za6-nbG4ZoUMXMpUK8KlCIosvQB0f-oq3k/edit?ts=5fd6c769#gid=907731238)

## Get started

* [How to install](./install.md)
* [Get Started](./docs/en/latest/getting-started.md)
* [Design introduction](./docs/en/latest/design.md)
* [FAQ](./docs/en/latest/FAQ.md)

## Prerequisites

Apisix ingress controller requires Kubernetes version 1.16+. Because we used `CustomResourceDefinition` v1 stable API.
From the version 1.0.0, APISIX-ingress-controller need to work with Apache APISIX version 2.7+.

## Works with APISIX Dashboard

Currently, APISIX Ingress Controller automatically manipulates some APISIX resources, which is not very compatible with APISIX Dashboard. In addition, users should not modify resources labeled `managed-by: apisix-ingress-controllers` via APISIX Dashboard.

## Internal Architecture

<img src="./docs/assets/images/apisix-ingress-controller-arch.png" alt="module" width="74.3%" height="55.9%" />

## Apache APISIX Ingress vs. Kubernetes Ingress Nginx

* Hot-reload during yaml apply.
* [More convenient canary deployment.](./docs/en/latest/concepts/apisix_route.md)
* Verify the correctness of the configuration, safe and reliable.
* [Rich plugins and ecology.](https://github.com/apache/apisix/tree/master/docs/en/latest/plugins)
* Supports APISIX custom resources and Kubernetes native Ingress resources.
* More active community

## Contributing

We welcome all kinds of contributions from the open-source community, individuals and partners.

* [Contributing Guide](./docs/en/latest/contribute.md)

### How to contribute

Most of the contributions that we receive are code contributions, but you can
also contribute to the documentation or simply report solid bugs
for us to fix.

 For new contributors, please take a look at issues with a tag called [Good first issue](https://github.com/apache/apisix-ingress-controller/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) or [Help wanted](https://github.com/apache/apisix-ingress-controller/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22).

### How to report a bug

* **Ensure the bug was not already reported** by searching on GitHub under [Issues](https://github.com/apache/apisix-ingress-controller/issues).

* If you're unable to find an open issue addressing the problem, [open a new one](https://github.com/apache/apisix-ingress-controller/issues/new). Be sure to include a **title and clear description**, as much relevant information as possible, and a **code sample** or an **executable test case** demonstrating the expected behavior that is not occurring.

### Contributor over time

[![Contributor over time](https://contributor-overtime-api.git-contributor.com/contributors-svg?chart=contributorOverTime&repo=apache/apisix-ingress-controller)](https://git-contributor.com/?chart=contributorOverTime&repo=apache/apisix-ingress-controller)

## Community

* Mailing List: Mail to dev-subscribe@apisix.apache.org, follow the reply to subscribe the mailing list.
* QQ Group - 578997126
* ![Twitter Follow](https://img.shields.io/twitter/follow/ApacheAPISIX?style=social) - follow and interact with us using hashtag `#ApacheAPISIX`
* [Bilibili video](https://space.bilibili.com/551921247)

## Todos

* More todos will display in [issues](https://github.com/apache/apisix-ingress-controller/issues?q=is%3Aopen+is%3Aissue+label%3Atriage%2Faccepted)

## User stories

- [aispeech: Why we create a new k8s ingress controller?(Chinese)](https://mp.weixin.qq.com/s/bmm2ibk2V7-XYneLo9XAPQ)
- [Tencent Cloud: Why choose Apache APISIX to implement the k8s ingress controller?(Chinese)](https://www.upyun.com/opentalk/448.html)

## Milestone

* [Milestone](https://github.com/apache/apisix-ingress-controller/milestones)

## Terminology

* Ingress APISIX: the whole service that contains the proxy ([Apache APISIX](https://apisix.apache.org)) and ingress controller (apisix ingress controller).
* apisix-ingress-controller: the ingress controller component.
