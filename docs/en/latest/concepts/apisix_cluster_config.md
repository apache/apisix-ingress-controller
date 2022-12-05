---
title: ApisixClusterConfig
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixClusterConfig
description: Guide to using ApisixClusterConfig custom Kubernetes resource.
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

`ApisixClusterConfig` is a Kubernetes CRD resource that can be used to describe an APISIX cluster and manage it.

## Monitoring

By default, monitoring is not enabled in an APISIX cluster. You can enable it by creating an `ApisixClusterConfig` resource.

The example below enabled [Prometheus](http://apisix.apache.org/docs/apisix/plugins/prometheus) and [SkyWalking](http://apisix.apache.org/docs/apisix/plugins/skywalking) for the "default" APISIX cluster (in [multi-cluster deployments](#multi-cluster-management)).

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
    skywalking:
      enable: true
      sampleRatio: 0.5
```

## Admin configuration

Instead of changing the deployment or pod definition files, you can use the `ApisixClusterConfig` resource to change the admin configurations.

The example below configures the base URL and admin key for the APISIX cluster "default":

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  admin:
    baseURL: http://apisix-gw.default.svc.cluster.local:9180/apisix/admin
    adminKey: "123456"
```

Once configured, other resources (Route, Upstream, etc) will be forwarded to the new address with the new admin key.

## Multi-cluster management

The `ApisixClusterConfig` resource can also be used to manage multiple APISIX clusters. This function is **not enabled currently** and it can only manage the cluster configured through `--default-apisix-cluster-name` attribute.

:::note

Deleting an `ApisixClusterConfig` resource will only reset the configurations of an APISIX cluster and will not affect its running.

:::
