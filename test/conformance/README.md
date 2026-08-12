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

# Gateway API conformance

This directory holds the [Gateway API conformance](https://gateway-api.sigs.k8s.io/docs/concepts/conformance/)
suite configuration. Running it produces a report that can be submitted to the
[Gateway API conformance reports](https://github.com/kubernetes-sigs/gateway-api/tree/main/conformance/reports)
repository, which requires the run to be reproducible by a third party.

## Prerequisites

The following binaries are assumed to be installed:

- [docker](https://docs.docker.com/get-started/get-docker/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kind](https://github.com/kubernetes-sigs/kind)
- [go](https://go.dev/learn/)

Tested on Linux. Any cluster works as long as it supports LoadBalancer
Services: the suite reaches the gateway through the data plane Service's
external address, and the controller publishes that address in every Gateway's
`status.addresses`. Steps 2 and 3 below only exist to give a local kind cluster
that capability.

## Reproduce

1. Clone the repository and check out the release to reproduce

   ```shell
   git clone https://github.com/apache/apisix-ingress-controller.git && cd apisix-ingress-controller
   git checkout <release>
   ```

   Checking out the release tag is what selects the published images for it and
   makes the report name that release, so the steps below are available from the
   first release that carries them. `make conformance-images` prints the three
   images the run will deploy; do not run `make build-image` here, it would
   replace a published image locally with a build of your own.

2. Create the cluster

   ```shell
   make kind-up
   ```

3. Run a local LoadBalancer provider

   ```shell
   make kind-lb
   ```

   This runs [cloud-provider-kind](https://kind.sigs.k8s.io/docs/user/loadbalancer)
   in the background, logging to `/tmp/cloud-provider-kind.log`. It needs access
   to the Docker socket. Skip this step on a cluster that already has
   LoadBalancer support.

4. Install the Gateway API and the controller's CRDs

   ```shell
   make install
   ```

   On a commit that is not a release, the run uses the `dev` images and the
   report names the commit. Build and load them into the cluster first:

   ```shell
   make build-image && make kind-load-images
   ```

5. Run the suite

   ```shell
   make conformance-test
   ```

   For the standalone data plane mode, run it with the provider type and the
   matching report mode:

   ```shell
   PROVIDER_TYPE=apisix-standalone make conformance-test CONFORMANCE_MODE=apisix-standalone
   ```

6. Read the report

   ```shell
   cat "$(make -s conformance-report-path)"
   ```

   Pass the same `CONFORMANCE_MODE` to get the path of a standalone run.

   The file is named `<channel>-<version>-<mode>-report.yaml`, which is the name
   the upstream repository expects, so it can be submitted as produced. Reports
   must be uploaded unmodified.
