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

# Table of Contents

- [0.5.0](#050)
- [0.4.0](#040)
- [0.3.0](#030)
- [0.2.0](#020)
- [0.1.0](#010)

# 0.5.0

A lot of important features are supported in this release, it makes apisix-ingress-controller more powerful and flexible.
Also, several bugs are fixed so the robustness is also enhanced.

We recommend you to use [Apache APISIX 2.5](https://github.com/apache/apisix/releases/tag/2.5) with this release. Note since CRDs are updated, when
you upgrade your old release, **manual steps are required to apply the new ApisixRoute**. Please see the instruction `7` in [FAQ](https://github.com/apache/apisix-ingress-controller/blob/master/docs/en/latest/FAQ.md) for more details.

## Core

* Support traffic split feature ([#308](https://github.com/apache/apisix-ingress-controller/pull/308))
* Support route match exprs ([#304](https://github.com/apache/apisix-ingress-controller/pull/304), [#306](https://github.com/apache/apisix-ingress-controller/pull/306))
* Support to configure [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resources in version `extensions/v1beta1` ([#296](https://github.com/apache/apisix-ingress-controller/pull/296), [#315](https://github.com/apache/apisix-ingress-controller/pull/315))
* Add name fields when generating APISIX Routes and Upstreams ([#333](https://github.com/apache/apisix-ingress-controller/pull/333))
* Support to use remote addrs as route match conditions ([#347](https://github.com/apache/apisix-ingress-controller/pull/347))
* Schema for ApisixRoute CRD ([#345](https://github.com/apache/apisix-ingress-controller/pull/345))

## Fix

* Sometimes ApisixRoute update is ineffective ([#319](https://github.com/apache/apisix-ingress-controller/pull/319))
* Priority field is not passed to APISIX ([#329](https://github.com/apache/apisix-ingress-controller/pull/329))
* Route rule name in ApisixRoute can be duplicated ([#330](https://github.com/apache/apisix-ingress-controller/pull/330))
* Use `PUT` instead of `PATCH` method when updating resources ([#353](https://github.com/apache/apisix-ingress-controller/pull/353))
* Secrets controller doesn't push the newest cert and priv key to APISIX ([#337](https://github.com/apache/apisix-ingress-controller/pull/337))

## Test

* Use [Kind](https://kind.sigs.k8s.io/) to run e2e suites ([#331](https://github.com/apache/apisix-ingress-controller/pull/331))
* Add e2e test cases for plugins redirect, uri-blocker, fault-injection, request-id, limit-count, echo, cors, response-rewrite, proxy-rewrite ([#320](https://github.com/apache/apisix-ingress-controller/pull/320), [#327](https://github.com/apache/apisix-ingress-controller/pull/327), [#328](https://github.com/apache/apisix-ingress-controller/pull/328), [#334](https://github.com/apache/apisix-ingress-controller/pull/334), [#336](https://github.com/apache/apisix-ingress-controller/pull/336), [#342](https://github.com/apache/apisix-ingress-controller/pull/342), [#341](https://github.com/apache/apisix-ingress-controller/pull/341))

# 0.4.0

This release mainly improves the program robustness and adds some features.

## Core

- Support Kubernetes Ingress resources [#250](https://github.com/apache/apisix-ingress-controller/pull/250)
- Support ApisixRoute v2alpha1 [#262](https://github.com/apache/apisix-ingress-controller/pull/262)
- Support healthchecks definition [#117](https://github.com/apache/apisix-ingress-controller/issues/117)
- Support secret controller [#284](https://github.com/apache/apisix-ingress-controller/pull/284)
- Project optimization [#92](https://github.com/apache/apisix-ingress-controller/issues/92)

## Test

- Add test cases for pkg/kube [#99](https://github.com/apache/apisix-ingress-controller/issues/99)

# 0.3.0

This release mainly improves the program robustness and adds some features.

## Core

- Support Leader election to let only the leader process resources [#173](https://github.com/apache/apisix-ingress-controller/pull/173);
- Let Controller itself generates resource ids instead of relying on APISIX [#199](https://github.com/apache/apisix-ingress-controller/pull/199);
- Change go module name from `github.com/api7/ingress-controller` to `github.com/apache/apisix-ingress-controller` [#220](https://github.com/apache/apisix-ingress-controller/pull/220);
- Re draw the design diagram [#214](https://github.com/apache/apisix-ingress-controller/pull/214);
- Support gRPC scheme in ApisixUpstream [#225](https://github.com/apache/apisix-ingress-controller/pull/225);
- SSL resource cache optimization [#203](https://github.com/apache/apisix-ingress-controller/pull/203);

## Deploy

- Complete the compatibility tests on Amazon EKS, Google Cloud GKE, Ali Cloud ACK and etc [#177](https://github.com/apache/apisix-ingress-controller/pull/177), [#180](https://github.com/apache/apisix-ingress-controller/pull/180), [#183](https://github.com/apache/apisix-ingress-controller/pull/183);
- Refactor the helm charts, merging ingress-apisix and ingress-apisix-base into apisix-ingress-controller [#213](https://github.com/apache/apisix-ingress-controller/pull/213);

## Test

- Now CI runs e2e test suites in parallel [#172](https://github.com/apache/apisix-ingress-controller/pull/172);

# 0.2.0

This release mainly improve basic features, bugfix and adds test cases.

## Core

- Enhanced documentation, easier to read and execute [#129](https://github.com/apache/apisix-ingress-controller/pull/129)
- API specification for CRDs [#151](https://github.com/apache/apisix-ingress-controller/pull/151)
- Support Canary plugin (Base on the [feature](https://github.com/apache/apisix/pull/2935) in Apache APISIX) [#13](https://github.com/apache/apisix-ingress-controller/issues/13)
- Support prometheus metrics [#143](https://github.com/apache/apisix-ingress-controller/pull/143)
- Support install ingress controller by Helm Chart [#153](https://github.com/apache/apisix-ingress-controller/pull/153)
- Support reconcile loop. [#149](https://github.com/apache/apisix-ingress-controller/pull/149) [#157](https://github.com/apache/apisix-ingress-controller/pull/157) [#163](https://github.com/apache/apisix-ingress-controller/pull/163)
- Support namespaces filtering. [#162](https://github.com/apache/apisix-ingress-controller/pull/162)
- Some Refactor. [#147](https://github.com/apache/apisix-ingress-controller/pull/147) [#155](https://github.com/apache/apisix-ingress-controller/pull/155) [#134](https://github.com/apache/apisix-ingress-controller/pull/134)

## Test case

more e2e case [#156](https://github.com/apache/apisix-ingress-controller/pull/156) [#142](https://github.com/apache/apisix-ingress-controller/pull/142)

[Back to TOC](#table-of-contents)

# 0.1.0

This release mainly improve basic features, bugfix and adds test cases.

## Core

- Enriched documentation.
- CI Integration. [#75](https://github.com/apache/apisix-ingress-controller/pull/75) [#80](https://github.com/apache/apisix-ingress-controller/pull/80) [#84](https://github.com/apache/apisix-ingress-controller/pull/84) [#87](https://github.com/apache/apisix-ingress-controller/pull/87) [#89](https://github.com/apache/apisix-ingress-controller/pull/89) [#97](https://github.com/apache/apisix-ingress-controller/pull/97)
- Support retry when sync failed. [#103](https://github.com/apache/apisix-ingress-controller/pull/103)
- Support using kustomize install all resources. [#72](https://github.com/apache/apisix-ingress-controller/pull/72)
- Support command line configuration. [#61](https://github.com/apache/apisix-ingress-controller/pull/61)
- Support to define SSL by CRD. [#95](https://github.com/apache/apisix-ingress-controller/pull/95)

## Test case

- Add E2E test environment. [#101](https://github.com/apache/apisix-ingress-controller/pull/101)

## Bugfix

- invalid memory address or nil pointer dereference. [#9](https://github.com/api7/seven/pull/9)

[Back to TOC](#table-of-contents)
