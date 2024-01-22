---
title: MTLS bypass based on regular expression matching against URI
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

APISIX allows configuring an URI whitelist to bypass MTLS. If the URI of a request is in the whitelist, then the client certificate will not be checked. Note that other URIs of the associated SNI will get HTTP 400 response instead of alert error in the SSL handshake phase, if the client certificate is missing or invalid.

::: note
This feature is only available in APISIX version 3.4 and above.
:::

The below example creates an APISIX ssl resource where MTLS is bypassed for any route that starts with `/ip`.

```yaml
apiVersion: %s
kind: ApisixTls
metadata:
  name: my-tls
spec:
  hosts:
  - httpbin.org
  secret:
    name: my-secret
    namespace: default
  client:
    caSecret:
      name: ca-secret
      namespace: default
    depth: 10
    skip_mtls_uri_regex:
    - /ip.*
```
