---
title: Upgrade Guide
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

## Version change

### ***1.6.0***

APISIX 3.x.x has changed the Admin API. To make APISIX Ingress compatible with these changes, you need to select the corresponding chart version. Please refer [1.5 to 1.6](#15-to-16).

### ***1.5.0***

- CRD has been upgraded to the V2 version, and V2beta3 has been marked as deprecated.
- `app_namespace` is deprecated. you can use `namespace_selector` instead.

### ***1.4.0***

- CRD unified upgrade to V2beta3, delete resource v2alpha1 and v1 versions

## Upgrade using Helm chart

Before upgrading APISIX Ingress, you need to update the corresponding CRD resource first, k8s will automatically replace it with the default CRD resource version, incompatible items will be discarded, and its configuration needs to be updated to the current version.

:::note

It is recommended not to upgrade across versions.

:::

### Operating Step

1. Update Helm repo

Before upgrading, you need to update the helm repo to ensure that the resources in the repo are up to date.

```sh
helm repo update
```

2. Upgrade CRDs

When the CRD exists, Helm Chart will not automatically update the CRD when upgrading or installing, so you need to update the CRD resource yourself

- Using Helm (Helm version >= 3.7.0)

```sh
helm show crds apisix/apisix-ingress-controller | kubectl apply -f -
```

> If the Helm version does not support it, you need to obtain it from the [apisix-helm-chart](https://github.com/apache/apisix-helm-chart) repository.
> CRDs directory: `charts/apisix-ingress-controller/crds/customresourcedefinitions.yaml`
>
> ```sh
> kubectl apply -f  https://raw.githubusercontent.com/apache/apisix-helm-chart/apisix-1.1.0/charts/apisix-ingress-controller/crds/customresourcedefinitions.yaml
> ```

3. Upgrade APISIX Ingress

Just as an example, the specific configuration needs to be modified by yourself. If you want to upgrade to a specific chart version, please add this flag `--version x.x.x`, Please refer [compatible-upgrade](#compatible-upgrade) or [incompatible-upgrade](#incompatible-upgrade).

```sh
helm upgrade apisix apisix/apisix \
  --set service.type=NodePort \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix
```

### Compatible upgrade

Compatible upgrades can be made without changing any resources.

#### ***1.5 to 1.6***

:::note

[Relevant version information and compatibility of apisix-helm-chart](https://github.com/apache/apisix-helm-chart#compatibility-matrix).

If you use the `apisix-ingress-controller` chart, you need to focus on the configuration item  [adminAPIVersion](https://github.com/apache/apisix-helm-chart/blob/apisix-ingress-controller-0.11.3/charts/apisix-ingress-controller/values.yaml#L134).

:::

You need to select the corresponding chart version according to the APISIX version as shown to install or upgrade:

|Chart version| APISIX version | Values |
|--| ---|--|
|apisix-1.1.0| >= 3.0.0 | |
|apisix-0.13.0| <= 2.15.x | |
|apisix-ingress-controller-0.11.3| >= 3.0.0| adminAPIVersion=v3 |
|apisix-ingress-controller-0.11.3| <= 2.15.x| |

For `APISIX:3.x.x`, Use `apisix` chart to upgrade:

```sh
helm upgrade apisix apisix/apisix --version 1.1.0 ***  # omit some configuration
```

For `APISIX:3.x.x`, use `apisix-ingress-controller` chart to upgrade:

```sh
helm upgrade apisix apisix/apisix-ingress-controller \
  --version 0.11.3 \
  --set config.apisix.adminAPIVersion=v3 # omit some configuration
```

#### ***1.4 to 1.5***

The chart version corresponding to `apisix-ingress-controller:1.5`:

* `apisix-0.11.3`
* `apisix-ingress-controller-0.10.1`

```sh
helm upgrade apisix apisix/apisix --version 0.11.3 ***  # omit some configuration
```

### Incompatible upgrade

#### ***1.3 to 1.4***

The chart version corresponding to `apisix-ingress-controller:1.4`:

* `apisix-0.10.2`
* `apisix-ingress-controller-0.9.3`

```sh
helm upgrade apisix apisix/apisix --version 0.10.2 ***  # omit some configuration
```

Incompatible upgrade, need to change resources.
ApisixRoute `object(http[].backend)` has been removed in V2beta3 and needs to be converted to `array(http[].backends)`. It is recommended not to upgrade across major versions.

## Validate Compatibility

Apache APISIX Ingress project is a continuously actively developed project.
In order to make it better, some broken changes will be added when the new version is released.
For users, how to upgrade safely becomes very important.

The policy directory of this project contains these compatibility check strategies,
you can use the [`conftest`](https://github.com/open-policy-agent/conftest) tool to check the compatibility.

Here's a quick example.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /ip
     exprs:
     - subject:
         scope: Header
         name: X-Foo
       op: Equal
       value: bar
   backends:
   - serviceName: httpbin
     servicePort: 80
```

It uses the `spec.http.backend` field that has been removed.
Save as httpbin-route.yaml.

Use conftest for compatibility check.

```bash
$ conftest test httpbin-route.yaml
FAIL - httpbin-route.yaml - main - ApisixRoute/httpbin-route: rule1 field http.backend has been removed, use http.backends instead.

2 tests, 1 passed, 0 warnings, 1 failure, 0 exceptions
```

Incompatible parts will generate errors.
