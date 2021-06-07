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

When using the APISIX ingress declarative configuration, we often use the `kubectl apply` command. If the configuration is verified by the schema and validation webhook, the configuration will be accepted by Kubernetes.

When the Ingress Controller watches the resource change, the logic unit of the Ingress Controller has just started to work. 
In various operations of the Ingress Controller, object conversion and more verification will be performed. 
When the verification fails, the Ingress Controller will throw an error message and will continue to retry 
until the declared state is successfully synchronized to APISIX.

Therefore, after the declarative configuration is accepted by Kubernetes, it does not mean that the configuration is synchronized to APISIX.

In this practice, we will show how to  check the status of CRD.

## Prerequisites

- an available Kubernetes cluster
- an available APISIX and APISIX Ingress Controller installation

## deploy and check ApisixRoute resource

