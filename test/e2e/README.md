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

apisix ingress controller e2e test suites
=========================================

For running e2e test cases, a Kubernetes cluster is needed, [minikube](https://minikube.sigs.k8s.io/docs/start/) is a good choice to build k8s cluster in development environment.

## Scaffold

a e2e test scaffold is prepared to run test cases easily. The source codes are in directory `test/e2e/scaffold`, it builds necessary running environment:

* Create a brand new namespace;
* Create etcd deployment and etcd service in the specified namespace;
* Create apisix deployment and apisix service in the specified namespace (note both the control plane and data plane are created);
* Create apisix-ingress-controller deployment in the specified namespace;

The abovementioned steps are run before each case starts and all resources will be destroyed after the case finishes.
