---
title: ApisixTls
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

ApisixTls associates with a Kubernetes [Secret](https://kubernetes.io/docs/concepts/configuration/secret/) resource and
generates an [APISIX SSL](http://apisix.apache.org/docs/apisix/admin-api#ssl) object. It asks the
Secret must have two keys `cert` and `key`, which used to store the certificate and private key in
PEM format respectively.

```shell
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

Note the `hosts` field should be written carefully, it's used by Apache APISIX to match the
correct certificate, what's more, it also should be matched with the [Server Name Indication](https://www.globalsign.com/en/blog/what-is-server-name-indication#:~:text=Server%20Name%20Indication%20(SNI)%20allows,in%20the%20CLIENT%20HELLO%20message)
extension in TLS, or the TLS handshaking might fail.

The apisix-ingress-controller will watch Secret resources that referred by ApisixTls objects, once a
Secret changed, apisix-ingress-controller will re translate all referred ApisixTls objects, converting them to APISIX SSL resources ultimately.
