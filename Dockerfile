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
ARG ENABLE_PROXY=false
ARG BASE_IMAGE_TAG=nonroot

FROM golang:1.20 AS build-env
LABEL maintainer="gxthrj@163.com"

WORKDIR /build
COPY go.* ./

RUN if [ "$ENABLE_PROXY" = "true" ] ; then go env -w GOPROXY=https://goproxy.cn,direct ; fi \
    && go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build make build

FROM gcr.io/distroless/static-debian12:${BASE_IMAGE_TAG}
LABEL maintainer="gxthrj@163.com"
ENV TZ=Hongkong

WORKDIR /ingress-apisix
COPY --from=build-env /build/apisix-ingress-controller .
COPY ./conf/apisix-schema.json ./conf/apisix-schema.json

ENTRYPOINT ["/ingress-apisix/apisix-ingress-controller", "ingress", "--config-path", "/ingress-apisix/conf/config.yaml"]
