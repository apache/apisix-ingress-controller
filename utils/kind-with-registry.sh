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

set -o errexit
set -o nounset
set -o pipefail

# desired cluster name; default is "apisix"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-apisix}"
K8S_VERSION=${K8S_VERSION:-v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6}
REPOSITORY=${REPOSITORY:-localhost}

if kind get clusters | grep -q ^apisix$ ; then
  echo "cluster already exists, moving on"
  exit 0
fi

# create registry container unless it already exists
kind_version=$(kind version)
kind_network='kind'
reg_name='kind-registry'
reg_port="${1:-5000}"
case "${kind_version}" in
  "kind v0.7."* | "kind v0.6."* | "kind v0.5."*)
    kind_network='bridge'
    ;;
esac

# create registry container unless it already exists
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run \
    -d --restart=always -p "${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

reg_host="${reg_name}"
if [ "${kind_network}" = "bridge" ]; then
    reg_host="$(docker inspect -f '{{.NetworkSettings.IPAddress}}' "${reg_name}")"
fi
echo "Registry Host: ${reg_host}"

# create a cluster with the local registry enabled in containerd
kind_node_image="kindest/node:${K8S_VERSION}"
echo "Kubernetes version: ${kind_node_image}"
cat <<EOF | kind create cluster --name "${KIND_CLUSTER_NAME}" --image ${kind_node_image} --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."${REPOSITORY}:${reg_port}"]
    endpoint = ["http://${reg_host}:5000"]
EOF

for node in $(kind get nodes --name "${KIND_CLUSTER_NAME}"); do
  kubectl annotate node "${node}" tilt.dev/registry=${REPOSITORY}:${reg_port};
done

if [ "${kind_network}" != "bridge" ]; then
  containers=$(docker network inspect ${kind_network} -f "{{range .Containers}}{{.Name}} {{end}}")
  needs_connect="true"
  for c in $containers; do
    if [ "$c" = "${reg_name}" ]; then
      needs_connect="false"
    fi
  done
  if [ "${needs_connect}" = "true" ]; then               
    docker network connect "${kind_network}" "${reg_name}" || true
  fi
fi
