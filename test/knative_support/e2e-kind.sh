#!/usr/bin/env bash
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
# This script runs e2e tests in a local kind environment.
#

set -euo pipefail

APISIX_NAMESPACE=ingress-apisix

export KO_DOCKER_REPO=kind.local
export KIND_CLUSTER_NAME="ingress-apisix-knative"
./test/knative_support/upload-test-images.sh

echo ">> Setup test resources"
ko apply -f test/knative_support/config

export NODE_IP=$(kubectl get nodes --namespace ${APISIX_NAMESPACE} -o jsonpath="{.items[0].status.addresses[0].address}")
echo
export "GATEWAY_OVERRIDE=apisix"
export "GATEWAY_NAMESPACE_OVERRIDE=${APISIX_NAMESPACE}"

echo ">> Running conformance tests"
# timeout is 4m to failfast since e2e test always timed out
go test -count=1 -short -timeout=4m -tags=e2e -test.v ./test/knative_support/conformance/... ./test/knative_support/e2e/... \
  --ingressendpoint=${NODE_IP} \
  --ingressClass=apisix
