# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

ARG BASE_IMAGE_TAG=nonroot

FROM debian:bullseye-slim AS deps
WORKDIR /workspace

ARG TARGETARCH
ARG ADC_VERSION

RUN apt update \
    && apt install -y wget \
    && wget https://github.com/api7/adc/releases/download/v${ADC_VERSION}/adc_${ADC_VERSION}_linux_${TARGETARCH}.tar.gz -O adc.tar.gz \
    && tar -zxvf adc.tar.gz \
    && mv adc /bin/adc \
    && rm -rf adc.tar.gz \
    && apt autoremove -y wget

FROM gcr.io/distroless/cc-debian12:${BASE_IMAGE_TAG}

ARG TARGETARCH

WORKDIR /app

COPY --from=deps /bin/adc /bin/adc
COPY ./bin/apisix-ingress-controller_${TARGETARCH} ./apisix-ingress-controller
COPY ./config/samples/config.yaml ./conf/config.yaml

ENTRYPOINT ["/app/apisix-ingress-controller"]
CMD ["-c", "/app/conf/config.yaml"]
