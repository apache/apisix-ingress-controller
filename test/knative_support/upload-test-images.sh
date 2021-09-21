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

function upload_test_images() {
  echo ">> Publishing test images"
  (
    # Script needs to be executed from repo root
    cd "$( dirname "$0")/../.."
#    local image_dir="$(find "${GOPATH}/pkg/mod/knative.dev" -type d -name "networking*" | head -n1)/test/test_images"
    local image_dir="$HOME/go/src/knative/networking/test/test_images"
    local docker_tag=$1
    local tag_option=""
    if [ -n "${docker_tag}" ]; then
      tag_option="--tags $docker_tag,latest"
    fi

    for yaml in $(find ${image_dir} -name '*.yaml'); do
      # Rewrite image reference to use vendor.
      #sed "s@knative.dev/networking@fhuzero/apisix-ingress-controller/tmpvendor/knative.dev/networking@g" $yaml \
      cat $yaml \
        `# ko resolve is being used for the side-effect of publishing images,` \
        `# so the resulting yaml produced is ignored.` \
        | ko resolve ${tag_option} -RBf- > /dev/null
    done
  )
}

: ${KO_DOCKER_REPO:?"You must set 'KO_DOCKER_REPO', see DEVELOPMENT.md"}

upload_test_images $@
