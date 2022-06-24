---
title: ApisixTls/v2 Reference
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

|     Field                     |  Type    | Description                                                                                                   |
|-------------------------------|----------|---------------------------------------------------------------------------------------------------------------|
| hosts                         | array    | The domain list to identify which hosts (matched with SNI) can use the TLS certificate stored in the Secret.  |
| secret                        | object   | The definition of the related Secret object with current ApisixTls object.                                    |
| secret.name                   | string   | The name of the related Secret object with current ApisixTls object.                                          |
| secret.namespace              | string   | The namespace of the related Secret object with current ApisixTls object.                                     |
| client                        | object   | The configuration of the certificate provided by the client.                                                  |
| client.caSecret               | object   | The definition of the related Secret object with the certificate provided by the client.                      |
| client.caSecret.name          | string   | The name of the related Secret object with the certificate provided by the client.                            |
| client.caSecret.namespace     | string   | The namespace of the related Secret object with the certificate provided by the client.                       |
| client.depth                  | int      | The max certificate of chain length.                                                                          |
