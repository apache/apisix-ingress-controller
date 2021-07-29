---
title: ApisixClusterConfig
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

`ApisixClusterConfig` is a CRD resource which used to describe an APISIX cluster, currently it's not a required
resource but its existence can enrich an APISIX cluster, for instance, enabling cluster-wide monitoring, rate limiting and so on.

monitoring features like collecting [Prometheus](https://prometheus.io/) metrics
and [skywalking](https://skywalking.apache.org/) spans

Monitoring
----------

By default, monitoring are not enabled for the APISIX cluster, this is not favorable
if you'd like to learn the real running status of your cluster. In such a case, you
could create a `ApisixClusterConfig` to enable these features explicitly.

```yaml
apiVersion: apisix.apache.org/v2alpha1
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

The above example enables both the Prometheus and Skywalking for the APISIX cluster which name is "default".
Please see [Prometheus in APISIX](http://apisix.apache.org/docs/apisix/plugins/prometheus) and [Skywalking in APISIX](http://apisix.apache.org/docs/apisix/plugins/skywalking) for the details.

Admin Config
------------

The default APISIX cluster is configured through command line options like `--default-apisix-cluster-xxx`. They are constant in apisix-ingress-controller's lifecycle, you have to change the definition
of Deployment or Pod template. Now with the help of `ApisixClusterConfig`, you can change some administrative fields on it.

```yaml
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  admin:
    baseURL: http://apisix-gw.default.svc.cluster.local:9180/apisix/admin
    adminKey: "123456"
```

The above `ApisixClusterConfig` sets the base url and admin key for the APISIX cluster `"default"`. Once this
resource is processed, resources like Route, Upstream and others will be pushed to the new address with the new admin key (for authentication).

Multiple Clusters Management
----------------------------

`ApisixClusterConfig` is also designed for supporting multiple clusters management, but currently this function IS NOT ENABLED YET.
Only the `ApisixClusterConfig` with the same named configured in `--default-apisix-cluster-name` option will be handled by apisix-ingress-controller, other instances will be neglected.

The current delete event for `ApisixClusterConfig` doesn't mean the apisix-ingress-controller will lose the view of the corresponding APISIX cluster but
resetting all the features on it, so the running of APISIX cluster is not influenced by this event.
