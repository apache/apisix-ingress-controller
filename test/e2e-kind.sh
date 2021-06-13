#!/usr/bin/env bash

# This script runs e2e tests on a local kind environment.

set -euo pipefail

APISIX_GATEWAY_NAMESPACE=ingress-apisix

export KO_DOCKER_REPO=kind.local
export KIND_CLUSTER_NAME="ingress-apisix-integration"
$(dirname $0)/upload-test-images.sh

echo ">> Setup test resources"
# TODO
ko apply -f test/config

#ip=$(kubectl get nodes -lkubernetes.io/hostname!=kind-control-plane -ojsonpath='{.items[*].status.addresses[?(@.type=="InternalIP")].address}' | head -n1)
export NODE_IP=$(kubectl get nodes --namespace ingress-apisix -o jsonpath="{.items[0].status.addresses[0].address}")
export NODE_PORT=$(kubectl get --namespace ingress-apisix -o jsonpath="{.spec.ports[0].nodePort}" services apisix-admin)
echo
export APISIX_IP=${NODE_IP}:${NODE_PORT}
export "GATEWAY_OVERRIDE=apisix"
export "GATEWAY_NAMESPACE_OVERRIDE=${APISIX_GATEWAY_NAMESPACE}"

echo ">> Running conformance tests"
# timeout is 1m to failfast since the test always timed out
go test -count=1 -short -timeout=1m -tags=e2e -test.v ./test/conformance/... ./test/e2eknative/... \
  --ingressendpoint=${NODE_IP} \
  --ingressClass=apisix
