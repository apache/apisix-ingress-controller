---
title: FAQ
keywords:
- APISIX Ingress
- Apache APISIX
- Kubernetes Ingress
- Gateway API
- FAQ
description: This document provides answers to frequently asked questions (FAQ) when using APISIX Ingress Controller.
---

<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

This document provides answers to frequently asked questions (FAQ) when using APISIX Ingress Controller.

### How does Ingress Controller handle route priority across multiple resources?

In APISIX, a higher value indicates a higher route priority.

* **Ingress:** Does not support explicit route priority. Routes created using Ingress are assigned a default priority of 0, typically the lowest.
* **HTTPRoute:** Has a [38-bit priority](https://github.com/apache/apisix-ingress-controller/blob/master/internal/adc/translator/httproute.go#L428-L448). The priority calculation is dynamic and may change, making exact values difficult to predict.
* **APISIXRoute:** Can be assigned an explicit priority. To have a higher priority than an HTTPRoute, the value must exceed 549,755,813,887 (2^39 − 1).

### How do HTTPRoute filters interact with PluginConfig CRDs?

APISIX maps built-in Gateway API HTTPRoute filters to specific plugins:

* `RequestHeaderModifier` → `proxy-rewrite`
* `RequestRedirect` → `redirect`
* `RequestMirror` → `proxy-mirror`
* `URLRewrite` → `proxy-rewrite`
* `ResponseHeaderModifier` → `response-rewrite`
* `CORS` → `cors`
* `ExtensionRef` → user-defined plugin reference

When both filters and a PluginConfig CRD are applied:

* If filters are applied first, PluginConfig overrides any overlapping plugin settings.
* If PluginConfig is applied first, filters merge with PluginConfig settings, and overlapping fields from filters take precedence.

### Can Admin API resources coexist with controller-managed resources?

With `provider.type: apisix`, synchronization selects resources labeled
`manager-by: apisix-ingress-controller`. Resources created outside the controller
without this label are excluded from deletion, including when a Gateway or
IngressClass is deleted. Do not reuse controller-generated resource IDs for
externally managed resources: a matching ID can still be overwritten.

ADC filters top-level resources. Routes nested under a selected Service are
included with that Service, so externally managed routes should not share a
controller-managed Service. Global rules and plugin metadata do not support this
label-based isolation and remain subject to full reconciliation. The label also
does not distinguish separate controller installations sharing one APISIX instance.
This behavior does not apply to `apisix-standalone`, which manages a complete
configuration.

When upgrading, unlabeled historical resources with IDs still present in the
controller's desired configuration are updated and labeled during synchronization.
Unlabeled orphaned resources are preserved. Review their ownership before deleting
them or adding the controller label; adding the label allows the next synchronization
to delete resources absent from the controller's desired configuration.
