---
title: Best practice to avoid race condition between Kubelet and APISIX
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

## Prerequisite

While scaling down the number of pods in your upstream, few of the pods are marked as Terminating. On the node where the Pod is running: as soon as the kubelet sees that a Pod has been marked as terminating (a graceful shutdown duration has been set), the kubelet begins the local Pod shutdown process. At the same time as the kubelet is starting graceful shutdown of the Pod, the control plane evaluates whether to remove that shutting-down Pod from EndpointSlice (and Endpoints) objects, where those objects represent a Service with a configured selector. ReplicaSets and other workload resources no longer treat the shutting-down Pod as a valid, in-service replica.
Pods that shut down slowly should not continue to serve regular traffic and should start terminating and finish processing open connections.

## The race condition

APISIX ingress controller also watches for pod updates(for upstream pods). And updates the APISIX instance with new upstreams and removes the older upstreams. But this process might take time depending on the latency between Ingress controller and APISIX instance. This latency can be large if APISIX and Ingress controller have a number of network hops between them or it can be low if you're running APISIX ingress controller in [composite mode](../composite.md) where both ingress controller and APISIX instance are running in the same pod. But there is some latency(t) and if the time taken by the Kubelet to terminate those pods is less than t then for some period of time users might get 5xx errors because few of the requests may get routed to the upstreams(terminated pods) that are no longer available.

## PreStop hook

If one of the Pod's containers has defined a `PreStop` hook and the `terminationGracePeriodSeconds` in the Pod spec is not set to 0, the kubelet runs that hook inside of the container. The default `terminationGracePeriodSeconds` setting is 30 seconds.

## Solution

You can add delay to the termination of the upstream pod by some seconds (for eg: 5 seconds) to make sure that the pod is not terminated before the APISIX instance gets updated.

:::note

If the preStop hook needs longer to complete than the default grace period allows, you must modify `terminationGracePeriodSeconds` to suit this. This is 30 seconds by default. For more information, refer to the [Kubernetes docs](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination).

:::

Below is the example usage of one such Pod configuration which will act as upstream for APISIX.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
spec:
  containers:
  - name: web-server
    image: web-server:latest
    lifecycle:
      preStop:
        exec:
          command: ["/bin/sh", "-c", "sleep 5"]

```
