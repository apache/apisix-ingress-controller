---
title: Checking the synchronization status of the CRDs
keywords:
  - APISIX Ingress
  - Apache APISIX
  - Kubernetes Ingress
  - APISIX CRDs
  - Synchronization
description: A guide to check the synchronization status of APISIX CRDs.
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

APISIX CRDs are applied to a Kubernetes cluster using the `kubectl apply` command. Behind the scenes, Kubernetes verifies the configuration using the [Open API V3 schema](https://swagger.io/specification/) and its validation webhooks (if any).

But this does not mean that the configuration is synchronized and validated by APISIX. APISIX will convert the declared configuration to APISIX-specific resources and verify it. If the verification fails, the Ingress controller will log an error message and will retry until the desired state is successfully synchronized to APISIX.

This guide will show how you can check the synchronization status of the CRDs.

## Example with ApisixRoute

This example uses [ApisixRoute](https://apisix.apache.org/docs/ingress-controller/references/apisix_route_v2) resources. But this also applies to other APISIX CRDs like [ApisixUpstream](https://apisix.apache.org/docs/ingress-controller/references/apisix_upstream), [ApisixTls](https://apisix.apache.org/docs/ingress-controller/references/apisix_tls_v2), and [ApisixClusterConfig](https://apisix.apache.org/docs/ingress-controller/references/apisix_cluster_config_v2).

We can deploy a sample ApisixRoute resource:

```yaml
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2
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
      backends:
        - serviceName: httpbin-service-e2e-test
          servicePort: 80
EOF
```

Once this resource is applied, you can check its synchronization status with its name as shown below:

```shell
kubectl describe ar httpbin-route
```

This will give the status as shown below:

```text title="output"
...
Status:
  Conditions:
    Message:              Sync Successfully
    Observed Generation:  1
    Reason:               ResourcesSynced
    Status:               True
    Type:                 ResourcesAvailable
Events:
  Type    Reason           Age                From           Message
  ----    ------           ----               ----           -------
  Normal  ResourcesSynced  50s (x2 over 50s)  ApisixIngress  ApisixIngress synced successfully
```

## Troubleshooting

If you are not able to see the status, please check if you are using:

1. An APISIX Ingress controller version `>=1.0`.
2. The [latest CRD definition file](https://github.com/apache/apisix-ingress-controller/tree/master/samples/deploy/crd/v1).
3. The latest [RBAC configuration](https://github.com/apache/apisix-ingress-controller/tree/master/samples/deploy/rbac).
