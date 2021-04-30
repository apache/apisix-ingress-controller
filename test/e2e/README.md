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

Scaffold
---------

a e2e test scaffold is prepared to run test cases easily. The source codes are in directory `test/e2e/scaffold`, it builds necessary running environment:

* Create a brand new namespace;
* Create etcd deployment and etcd service in the specified namespace;
* Create apisix deployment and apisix service in the specified namespace (note both the control plane and data plane are created);
* Create apisix-ingress-controller deployment in the specified namespace;
* Create a http server with [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/) as the upstream.

The above mentioned steps are run before each case starts and all resources will be destroyed after the case finishes.

Plugins
-------

Test cases inside `plugins` directory test the availability about APISIX plugins.

Features
--------

Test caes inside `features` directory test some features about APISIX, such as traffic-split, health check and so on.

Quick Start
-----------

Run `make e2e-test` to run the e2e test suites in your development environment, a several stuffs that this command will do:

1. Create a Kubernetes cluster by [kind](https://kind.sigs.k8s.io/), please installing in advance.
2. Build and push all related images to this cluster.
3. Run e2e test suites.

Step `1` and `2` can be skipped by passing `E2E_SKIP_BUILD=1` to this directive, also, you can customize the
running concurrency of e2e test suites by passing `E2E_CONCURRENCY=X` where `X` is the desired number of cases running in parallel.

Run `make kind-reset` to delete the cluster that created by `make e2e-test`.
