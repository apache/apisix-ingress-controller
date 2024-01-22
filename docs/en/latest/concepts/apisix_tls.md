---
title: ApisixTls
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixTls
description: Guide to using ApisixTls custom Kubernetes resource.
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

`ApisixTls` is a Kubernetes CRD object used to create an [APISIX SSL object](http://apisix.apache.org/docs/apisix/admin-api#ssl). It uses a [Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) with two keys, `cert` containing the certificate, and `key` containing the private key in PEM format.

See [reference](https://apisix.apache.org/docs/ingress-controller/references/apisix_tls_v2) for the full API documentation.

The example below shows how you can configure an `ApisixTls` resource:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: sample-tls
spec:
  hosts:
  - httpbin.org
  secret:
    name: htpbin-cert
    namespace: default
```

:::info IMPORTANT

Make sure that the `hosts` field is accurate. APISIX uses the `host` field to match the correct certificate. It should also match the [Server Name Indication](https://www.globalsign.com/en/blog/what-is-server-name-indication#:~:text=Server%20Name%20Indication%20(SNI)%20allows,in%20the%20CLIENT%20HELLO%20message) extension in TLS, or the TLS handshake might fail.

:::

APISIX Ingress will watch the secret resources referred by `ApisixTls` objects and re-translates it to APISIX resources if they are changed.

## Bypassing MTLS based on regular expression matching against URI

::: note
This feature is only supported with APISIX version 3.4 or above.
:::

APISIX allows configuring an URI whitelist to bypass MTLS. If the URI of a request is in the whitelist, then the client certificate will not be checked. Note that other URIs of the associated SNI will get HTTP 400 response instead of alert error in the SSL handshake phase, if the client certificate is missing or invalid.

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
