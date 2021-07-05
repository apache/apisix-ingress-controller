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

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")
PROJECT_ROOT="$SCRIPT_ROOT/.."
GENERATED_ROOT="$PROJECT_ROOT/.generated"

PKG_NAME="github.com/apache/apisix-ingress-controller"

# Make sure no pollution
rm -rf "$GENERATED_ROOT"
trap 'rm -rf "$GENERATED_ROOT"' EXIT

# Prepare existed codes
mkdir -p "${GENERATED_ROOT}/${PKG_NAME}/pkg/kube/apisix"
cp -r "${PROJECT_ROOT}/pkg/kube/apisix/client" "${GENERATED_ROOT}/${PKG_NAME}/pkg/kube/apisix"

cp_deepcopy() {
  local SRC_PATH="$1"
  local CP_SCRIPT='FILE_PATH=$1; TARGET_PATH=$2; DIR="$(dirname $FILE_PATH)"; mkdir -p $TARGET_PATH/$DIR  && cp $FILE_PATH $TARGET_PATH/$DIR'
  find "${PROJECT_ROOT}/$SRC_PATH" -type f -name 'zz_generated.deepcopy.go' \
    -exec sh -c "$CP_SCRIPT" _ {} "${GENERATED_ROOT}/${PKG_NAME}" ';'
}

cp_deepcopy "pkg/kube/apisix/"
cp_deepcopy "pkg/types/apisix/"

# Verify
ret=0

bash "${SCRIPT_ROOT}"/generate-groups.sh "deepcopy,client,informer,lister" \
  ${PKG_NAME}/pkg/kube/apisix/client ${PKG_NAME}/pkg/kube/apisix/apis \
  config:v1,v2alpha1,v2beta1 ${PKG_NAME} \
  --output-base "$GENERATED_ROOT" \
  --go-header-file "${SCRIPT_ROOT}"/boilerplate.go.txt \
  --verify-only || ret=$?


if [[ $ret -eq 0 ]]; then
bash "${SCRIPT_ROOT}"/generate-groups.sh "deepcopy" \
  ${PKG_NAME}/pkg/types ${PKG_NAME}/pkg/types \
  apisix:v1 ${PKG_NAME} \
  --output-base "$GENERATED_ROOT" \
  --go-header-file "${SCRIPT_ROOT}"/boilerplate.go.txt \
  --verify-only|| ret=$?
fi

if [[ $ret -eq 0 ]]; then
  echo "Generated codes up to date."
else
  echo "Generated codes out of date. Please run \`make codegen\`"
  exit 1
fi
