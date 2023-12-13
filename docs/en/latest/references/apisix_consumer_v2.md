---
title: ApisixConsumer/v2
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixConsumer
description: Reference for ApisixConsumer/v2 custom Kubernetes resource.
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

## Spec

See the [definition](../../../../samples/deploy/crd/v1/ApisixConsumer.yaml) on GitHub.

| Field            | Type    | Description                                                                                                                                    |
|------------------|---------|------------------------------------------------------------------------------------------------------------------------------------------------|
| authParameter          | object   | Configuration of one of the available authentication plugins.                                                                                                   |
| authParameter.basicAuth.value   | object   | Plugin configuration for [`basic-auth` plugin](https://apisix.apache.org/docs/apisix/plugins/basic-auth/)                                      |
| authParameter.basicAuth.secretRef.name   | string   | You can store plugin configuration in Kubernetes secret and reference here with the secret name                                     |
| authParameter.keyAuth.value   | object   | Plugin configuration for [`key-auth` plugin](https://apisix.apache.org/docs/apisix/plugins/key-auth/)
| authParameter.keyAuth.secretRef.name   | string   | You can store plugin configuration in Kubernetes secret and reference here with the secret name.                                    |
| authParameter.jwtAuth.value   | object   | Plugin configuration for [`jwt-auth` plugin](https://apisix.apache.org/docs/apisix/plugins/jwt-auth/)
| authParameter.jwtAuth.secretRef.name   | string   | You can store plugin configuration in Kubernetes secret and reference here with the secret name.                                    |
| authParameter.wolfRBAC.value   | object   | Plugin configuration for [`wolf-rbac` plugin](https://apisix.apache.org/docs/apisix/plugins/wolf-rbac/)
| authParameter.wolfRBAC.secretRef.name   | string   | You can store plugin configuration in Kubernetes secret and reference here with the secret name.                                    |
| authParameter.hmacAuth.value   | object   | Plugin configuration for [`hmac-auth` plugin](https://apisix.apache.org/docs/apisix/plugins/hmac-auth/)
| authParameter.hmacAuth.secretRef.name   | string   | You can store plugin configuration in Kubernetes secret and reference here with the secret name.                                    |
| authParameter.ldapAuth.value   | object   | Plugin configuration for [`ldap-auth` plugin](https://apisix.apache.org/docs/apisix/plugins/ldap-auth/)
| authParameter.ldapAuth.secretRef.name   | string   | You can store plugin configuration in Kubernetes secret and reference here with the secret name.                                    |
