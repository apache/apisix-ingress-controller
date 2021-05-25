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

# Make sure no pollution
rm -rf "$GENERATED_ROOT"

bash "${SCRIPT_ROOT}"/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis \
  config:v1,v2alpha1 github.com/apache/apisix-ingress-controller \
  --output-base "$GENERATED_ROOT" \
  --go-header-file "${SCRIPT_ROOT}"/boilerplate.go.txt

bash "${SCRIPT_ROOT}"/generate-groups.sh "deepcopy" \
  github.com/apache/apisix-ingress-controller/pkg/types github.com/apache/apisix-ingress-controller/pkg/types \
  apisix:v1 github.com/apache/apisix-ingress-controller \
  --output-base "$GENERATED_ROOT" \
  --go-header-file "${SCRIPT_ROOT}"/boilerplate.go.txt

cp -r "$GENERATED_ROOT/github.com/apache/apisix-ingress-controller/"** "$PROJECT_ROOT"
rm -rf "$GENERATED_ROOT"
