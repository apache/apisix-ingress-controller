---
title: ApisixConsumer
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixConsumer
description: Guide to using ApisixConsumer custom Kubernetes resource.
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

`ApisixConsumer` is a Kubernetes CRD object that provides a spec to create an APISIX consumer and uniquely identify using any of the available authentication plugins.

:::note

Currently only authentication plugins are supported with `ApisixConsumer` CRD.

:::

## Usage

The example below shows how you can configure an `ApisixConsumer` resource with `keyAuth` plugin:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: jack
spec:
  authParameter:
    keyAuth:
      value:
        key: "auth-one"
```

You can then enable `key-auth` plugin on a route and use this consumer.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
    - name: route-1
      match:
        hosts:
          - httpbin.org
        paths:
          - /*
      backends:
        - serviceName: httpbin
          servicePort: 80
      authentication:
        enable: true
        type: keyAuth
```

Now if you send a request without the API key, you will get a response with 401 Unauthorized.

```shell
curl http://127.0.0.1:9080/headers -H 'Host: httpbin.org'
```

It should output:

```shell
{"message":"Missing API key found in request"}
```

But if you pass the key as configured in the ApisixConsumer resource, the request passes.

```shell
curl -H "Host: httpbin.org" -H "apiKey: auth-one" http://127.0.0.1:9080/headers
```

It should output:

```json
{
  "headers": {
    "Accept": "*/*", 
    "Apikey": "auth-one", 
    "Host": "httpbin.org", 
    "User-Agent": "curl/8.1.2", 
    "X-Forwarded-Host": "httpbin.org"
  }
}
```

Similarly you can  use other authentication plugins with `ApisixConsumer`. See [reference](../references/apisix_consumer_v2.md) for the full API documentation.
