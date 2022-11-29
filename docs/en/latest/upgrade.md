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

## Upgrade using Helm chart

Note: Before upgrading APISIX Ingress, you need to update the corresponding CRD resource first, k8s will automatically replace it with the default CRD resource version, incompatible items will be discarded, and its configuration needs to be updated to the current version.

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
> Directory: `charts/apisix-ingress-controller/crds/customresourcedefinitions.yaml`
>
> ```sh
> kubectl apply -f  https://raw.githubusercontent.com/apache/apisix-helm-chart/apisix-0.11.1/charts/apisix-ingress-controller/crds/customresourcedefinitions.yaml
> ```

3. UpgradeAPISIX Ingress

Just as an example, the specific configuration needs to be modified by yourself.

```sh
helm upgrade apisix apisix/apisix \
  --set gateway.type=NodePort \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix
```

### Precautions

It is recommended not to upgrade across major versions.

#### 1.4 -> 1.5 -> 1.6

Compatible with upgrades without changing any resources.

#### 1.3 -> 1.4

Incompatible upgrade, need to change resources.
ApisixRoute object(http[].backend) has been removed in V2beta3 and needs to be converted to array(http[].backends). It is recommended not to upgrade across major versions.

## Version change

### 1.6.0

- No breaking changes in this release.

### 1.5.0

- CRD has been upgraded to the V2 version, and V2beta3 has been marked as deprecated.
- app_namespace e is deprecated, you can use namespace_selector instead.

### 1.4.0

- CRD unified upgrade to V2beta3, delete resource v2alpha1 and v1 versions

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
