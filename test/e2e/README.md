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
Sepecially, if test case failed and e2e-test run in dev mode, the related resources will not be destroyed. This fearure is useful for debugging.
Also, you can disable this feature by unset `E2E_ENV=dev`.

Plugins
-------

Test cases inside `plugins` directory test the availability about APISIX plugins.

Features
--------

Test case inside `features` directory test some features about APISIX, such as traffic-split, health check and so on.

Quick Start
-----------

Run `make e2e-test-local` to run the e2e test suites in your development environment, a several stuffs that this command will do:

1. Create a Kubernetes cluster by [kind](https://kind.sigs.k8s.io/), please installing in advance.
2. Build and push all related images to this cluster.
3. Run e2e test suites.

Run `make e2e-test` to run the e2e test suites in an existing cluster, you can specify custom registry by passing REGISTRY(eg docker.io).

Step `1` and `2` can be skipped by passing `E2E_SKIP_BUILD=1` to this directive, also, you can customize the
running concurrency of e2e test suites by passing `E2E_NODES=X` where `X` is the desired number of cases running in parallel.

You can run specific test cases by passing the environment variable `E2E_FOCUS=suite-<suite name>`, where `<suite name>` can be found under `test/e2e` directory.
For example, `E2E_FOCUS=suite-plugins* make e2e-test` will only run test cases in `test/e2e/suite-plugins` directory.

Run `make kind-reset` to delete the cluster that created by `make e2e-test`.

How to name test cases
-----------

Because we use `ginkgo --focus` option and the prefix `suite-<suite name>` to split test cases and make them run in parallel in CI, test cases should be named in the following way:

- All test cases are grouped by directories, and **their names should have `suite-` prefix**
- All top level specs (i.e. `ginkgo.Describe`) under the suite directory should have corresponding `suite-<suite-name>:` prefix.

Run `make names-check` to check the above naming convention.
