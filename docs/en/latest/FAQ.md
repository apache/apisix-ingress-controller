---
title: FAQ
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

### 1. How to bind Service and Upstream

All resource objects are uniquely determined by the namespace / name / port combination Id. If the combined Id is the same, the `service` and `upstream` will be considered as a binding relationship.

### 2. When modifying a CRD, how do other binding objects perceive it

This is a cascading update problem, see for details [apisix-ingress-controller Design ideas](./design.md)

### 3. Can I mix CRDs and admin api to define routing rules

No, currently we are implementing one-way synchronization, that is, CRDs file -> Apache AIPSIX. If the configuration is modified separately through admin api, it will not be synchronized to CRDs in Kubernetes.

This is because CRDs are generally declared in the file system, and Apply to enter Kubernetes etcd, we follow the definition of CRDs and synchronize to Apache Apisix Data Plane, but the reverse will make the situation more complicated.

### 4. Why there are some error logs like "list upstreams failed, err: http get failed, url: blahblahblah, err: status: 401"

So far apisix-ingress-controller doesn't support set admin_key for Apache APISIX, so when you deploy your APISIX cluster, admin_key should be removed from config.

Note since APISIX have two configuration files, the first is config.yaml, which contains the user specified configs, the other is config-default.yaml, which has all default items, config items in these two files will be merged. So admin_key in both files should be removed. You can customize these two configuration files and mount them to APISIX deployment.

### 5. Failed to create route with `ApisixRoute`

When `apisix-ingress-controller` creates a route with CRD, it checks the `Endpoint` resources in Kubernetes (matched by namespace_name_port). If the corresponding endpoint information is not found, the route will not be created and wait for the next retry.

Tips: The failure caused by empty upstream nodes is a limitation of Apache APISIX, related [issue](https://github.com/apache/apisix/issues/3072)

### 6. What is the retry rule of `apisix-ingress-controller`

If an error occurs during the process of `apisix-ingress-controller` parsing CRD and distributing the configuration to APISIX, a retry will be triggered.

The delayed retry method is adopted. After the first failure, it is retried once per second. After 5 retries are triggered, the slow retry strategy will be enabled, and the retry will be performed every 1 minute until it succeeds.

### 7. What if the CRDs need to be updated when you upgrading apisix-ingress-controller

CRDs upgrading is special as helm chart will skip to apply these resources when they already exist.

> With the arrival of Helm 3, we removed the old crd-install hooks for a more simple methodology. There is now a special directory called crds that you can create in your chart to hold your CRDs. These CRDs are not templated, but will be installed by default when running a helm install for the chart. If the CRD already exists, it will be skipped with a warning. If you wish to skip the CRD installation step, you can pass the --skip-crds flag.

In such a case, you may need to apply these CRDs by yourself.

```shell
kubectl apply -k samples/deploy/crd/
```

### 8. Why apisix-ingress-controller reports similar errors when running: no matches for kind "ApisixRoute" in version "apisix.apache.org/v2beta3"

The version `v2beta3` is the newest version of resource `ApisixRoute`. You need to check if the CRD definition file is up to date.

```shell
kubectl get crd apisixroutes.apisix.apache.org -o jsonpath='{ .spec.versions[*].name }' -A
```

If you can not find `v2beta3` in `ApisixRoute` definition file. Please apply the latest version of `ApisixRoute`.

Ref to FAQ #7.

### 9. Modify the Admin API key in APISIX-Ingress

Usually, you need to modify the Admin API key to protect Apache APISIX. Please refer to this [link](https://apisix.apache.org/docs/apisix/how-to-build/#updating-admin-api-key) to simply change Apache APISIX.

However, in apisix-ingress-controller, if we need to change the Admin API key, we also need to change the Admin API key in apisix-ingress-controller. There are two different ways to implement the requirements here.

For the first method, we need to modify the Admin API credentials values in both the `apisix/values.yaml` and `apisix/apisix-ingress-controller/values.yaml` files. You can refer to these two links(apisix's [values.yaml](https://github.com/apache/apisix-helm-chart/blob/57cdbe461765cd49af2195cc6a1976cc55262e9b/charts/apisix/values.yaml#L181) && apisix-ingress-controller's [values.yaml](https://github.com/apache/apisix-helm-chart/blob/57cdbe461765cd49af2195cc6a1976cc55262e9b/charts/apisix-ingress-controller/values.yaml#L128)).

Another method, you can just pass `--set ingress-controller.config.apisix.adminKey=<Your new admin key> --set admin.credentials.admin=<Your new admin key>`  to `helm install` command.
