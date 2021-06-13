#!/bin/bash

set -euo pipefail

KNATIVE_NAMESPACE=knative-serving
APISIX_GATEWAY_NAMESPACE=ingress-apisix

export KIND_CLUSTER_NAME="ingress-apisix-integration"
kind delete cluster
kind create cluster

echo "Deploying Knative Serving"
KNATIVE_VERSION=v0.23.0
kubectl apply -f https://github.com/knative/serving/releases/download/${KNATIVE_VERSION}/serving-crds.yaml
kubectl apply -f https://github.com/knative/serving/releases/download/${KNATIVE_VERSION}/serving-core.yaml
kubectl patch configmap/config-network -n ${KNATIVE_NAMESPACE} --type merge -p '{"data":{"ingress.class":"apisix"}}'

export KO_DOCKER_REPO=kind.local

# TODO
kubectl create namespace ${APISIX_GATEWAY_NAMESPACE}
echo "Deploying APISIX"
# Should already be installed by Helm, check kubectl to change service type to NodePort
helm install apisix apisix/apisix \
  --set admin.allow.ipList="{0.0.0.0/0}" \
  --namespace ${APISIX_GATEWAY_NAMESPACE}
kubectl patch svc apisix-admin --type='json' -p '[{"op":"replace","path":"/spec/type","value":"NodePort"}]' -n=${APISIX_GATEWAY_NAMESPACE}
echo "Wait for APISIX deployment to be up (timeout=300s)"
kubectl -n ${APISIX_GATEWAY_NAMESPACE} wait --timeout=300s --for=condition=Available deployments --all
#ip=$(kubectl get nodes -lkubernetes.io/hostname!=kind-control-plane -ojsonpath='{.items[*].status.addresses[?(@.type=="InternalIP")].address}' | head -n1)
#port=$(kubectl -n ingress-apisix get svc apisix -ojsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
export NODE_IP=$(kubectl get nodes --namespace ingress-apisix -o jsonpath="{.items[0].status.addresses[0].address}")
export NODE_PORT=$(kubectl get --namespace ingress-apisix -o jsonpath="{.spec.ports[0].nodePort}" services apisix-admin)
echo
export APISIX_IP=${NODE_IP}:${NODE_PORT}
echo "You can connect to APISIX Gateway at ${APISIX_IP}"
echo "Example usage: 'curl \"http://${APISIX_IP}/apisix/admin/services/\" -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1'"

echo "Deploying APISIX ingress controller"
# https://github.com/google/ko#does-ko-work-with-kustomize
# https://github.com/apache/apisix-ingress-controller/blob/master/install.md#kustomize-support

kubectl kustomize samples/deploy |
  sed 's/LoadBalancer/NodePort/g' |
  sed 's@http://127.0.0.1:9080/apisix/admin@http://${APISIX_IP}/apisix/admin@g' |
  ko resolve -f - |
  kubectl apply -f -

echo "Wait for APISIX ingress controller deployment to be up (timeout=300s)"
kubectl -n ${APISIX_GATEWAY_NAMESPACE} wait --timeout=300s --for=condition=Available deployments --all
