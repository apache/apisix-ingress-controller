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

This directory holds the [Gateway API conformance](https://gateway-api.sigs.k8s.io/concepts/conformance/)
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

1. Clone the repository and check out the version to test

   ```shell
   git clone https://github.com/apache/apisix-ingress-controller.git && cd apisix-ingress-controller
   git checkout 2.2.0
   ```

   The checked-out state selects the images: on a release tag the published
   images for that release are pulled, on any other commit the locally built
   `dev` images are used. Verify with:

   ```shell
   make print-CONFORMANCE_INGRESS_IMAGE print-CONFORMANCE_ADC_IMAGE print-CONFORMANCE_APISIX_IMAGE
   ```

2. Create the cluster

   ```shell
   make kind-up
   ```

3. Run a local LoadBalancer provider

   ```shell
   make conformance-lb
   ```

   This runs [cloud-provider-kind](https://kind.sigs.k8s.io/docs/user/loadbalancer)
   in the background, logging to `/tmp/cloud-provider-kind.log`. Skip this step
   on a cluster that already has LoadBalancer support.

4. Install the Gateway API and the controller's CRDs

   ```shell
   make install
   ```

5. Only when testing an unreleased commit, build the images and load them into
   the cluster. On a release tag skip this step; the published images are
   pulled instead.

   ```shell
   make build-image
   make kind-load-adc-image kind-load-ingress-image
   ```

6. Run the suite

   ```shell
   make conformance-test
   ```

   For the standalone data plane mode, run it with the provider type and the
   matching report mode:

   ```shell
   PROVIDER_TYPE=apisix-standalone make conformance-test CONFORMANCE_MODE=apisix-standalone
   ```

7. Read the report

   ```shell
   cat "$(make -s conformance-report-path)"
   ```

## What the run declares

The suite fills the report's `implementation` block and the report file name
from make variables, so nothing has to be edited by hand:

| variable | default | meaning |
| --- | --- | --- |
| `CONFORMANCE_ORGANIZATION` | `apache` | report `implementation.organization` |
| `CONFORMANCE_PROJECT` | `apisix-ingress-controller` | report `implementation.project` |
| `CONFORMANCE_URL` | repository page | report `implementation.url` |
| `CONFORMANCE_CONTACT` | issue tracker | report `implementation.contact` |
| `VERSION` | current release | report `implementation.version` |
| `CONFORMANCE_CHANNEL` | `experimental` | the channel `install-gateway-api` installs |
| `CONFORMANCE_MODE` | `default` | deployment mode, `apisix-standalone` for the standalone data plane |
| `CONFORMANCE_PROFILES` | `GATEWAY-HTTP,GATEWAY-GRPC,GATEWAY-TLS` | profiles to run |
| `SUPPORTED_EXTENDED_FEATURES` | see `Makefile` | extended features claimed |

The report is written to `<channel>-<version>-<mode>-report.yaml`, which is the
name the upstream repository expects, so the file can be submitted as produced.
Reports must be uploaded unmodified.

## Submitting a report

1. Push the release tag, then let the tag run of the `APISIX Conformance Test`
   workflow produce the reports, or run the steps above on the tag.
2. Download the `conformance-report-<provider>` artifacts.
3. Open a pull request against
   [kubernetes-sigs/gateway-api](https://github.com/kubernetes-sigs/gateway-api)
   adding the reports and a `README.md` under
   `conformance/reports/<gateway-api-version>/apache-apisix-ingress-controller/`.
   The README must carry a table of contents and a Reproduce section; upstream
   CI checks that it exists and that every link in it resolves.

## Skipped tests

`conformance_test.go` skips three groups of tests, each with the reason in a
comment next to it: certificate SAN handling, TLS passthrough, which APISIX
does not implement because it terminates TLS and matches stream routes by SNI,
and a set of known gaps tracked for follow-up. Skipped tests make the affected
profile `partial` rather than `success` in the report.
