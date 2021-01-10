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

- [0.2.0](#020)
- [0.1.0](#010)

# 0.2.0

This release mainly improve basic features, bugfix and adds test cases.

## Core

- Enhanced documentation, easier to read and execute[#129](https://github.com/apache/apisix-ingress-controller/pull/129)
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

- Add E2E test enviroment. [#101](https://github.com/apache/apisix-ingress-controller/pull/101)

## Bugfix

- invalid memory address or nil pointer dereference. [#9](https://github.com/api7/seven/pull/9)

[Back to TOC](#table-of-contents)
