---
title: How to quickly check the synchronization status of CRD
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

When using the Apache APISIX Ingress Controller declarative configuration, often use the `kubectl apply` command. Only if the configuration was verified by its [Open API V3 Schema definition](https://swagger.io/specification/) and its validation webhooks (if any), can the configuration be accepted by Kubernetes.

When the Apache APISIX Ingress Controller watches the resource change, the logic unit of the Apache APISIX Ingress Controller has just started to work.
In various operations of the Apache APISIX Ingress Controller, object conversion and more verification will be performed.
When the verification fails, the Apache APISIX Ingress Controller will log an error message and will continue to retry until the declared state is successfully synchronized to APISIX.

Therefore, after the declarative configuration is accepted by Kubernetes, it does not mean that the configuration is synchronized to APISIX.

In this practice, we will show how to check the status of CRD.

## Prerequisites

- an available Kubernetes cluster (version >= 1.14)
- an available Apache APISIX (version >= 2.6) and Apache APISIX Ingress Controller (version >= 0.6.0) installation

## Take ApisixRoute resource as an example

### deploy and check ApisixRoute resource

1. first deploy an ApisixRoute resource

e.g.

```yaml
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
    - name: rule1
      match:
        hosts:
          - httpbin.com
        paths:
          - /ip
      backend:
        serviceName: httpbin-service-e2e-test
        servicePort: 80
  EOF
```

2. After apply the ApisixRoute resource, now check the status of CRD

```shell
kubectl describe ar -n test httpbin-route
```

Then, will see the status of `httpbin-route` resource.

```yaml
...
Status:
  Conditions:
    Last Transition time:  2021-06-06T09:50:22Z
    Message:               Sync Successfully
    Reason:                ResourceSynced
    Status:                True
    Type:                  ReousrceReady
...
```

### Also supports checking the status of other resources

`ApisixUpstream`
`ApisixTls`
`ApisixClusterConfig`
`ApisixConsumer`

## Frequent Questions

If can not see the Status information, please check the following points:

1. The version of Apache APISIX Ingress Controller needs to be >= 1.0.
2. Use the latest CRD definition file, refer to [here](https://github.com/apache/apisix-ingress-controller/tree/master/samples/deploy/crd/v1beta1).
3. Use the latest RBAC configuration, refer to [here](https://github.com/apache/apisix-ingress-controller/tree/master/samples/deploy/rbac).
