---
title: ApisixTls/v2
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixTls
description: Reference for ApisixTls/v2 custom Kubernetes resource.
---
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

See [concepts](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_tls) to learn more about how to use the ApisixTls resource.

## Spec

See the [definition](https://github.com/apache/apisix-ingress-controller/blob/master/samples/deploy/crd/v1/ApisixTls.yaml) on GitHub.

| Attribute                 | Type   | Description                                                                             |
|---------------------------|--------|-----------------------------------------------------------------------------------------|
| hosts                     | array  | List of hosts (with matched SNI) that can use the TLS certificate stored in the Secret. |
| secret                    | object | Definition of the Secret related to the current `ApisixTls` object.                     |
| secret.name               | string | Name of the Secret related to the current `ApisixTls` object.                           |
| secret.namespace          | string | Namespace of the Secret related to the current `ApisixTls` object.                      |
| client                    | object | Configuration for the certificate provided by the client.                               |
| client.caSecret           | object | Definition of the Secret related to the certificate.                                    |
| client.caSecret.name      | string | Name of the Secret related to the certificate provided by the client.                   |
| client.caSecret.namespace | string | Namespace of the Secret related to the certificate.                                     |
| client.depth              | int    | The maximum length of the certificate chain.                                            |
| client.skip_mtls_uri_regex              | array    | List of uri regular expression to skip mtls.                                            |
