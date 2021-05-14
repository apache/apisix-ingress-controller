---
title: Annotations
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

This document describes all supported annotations and their functions. You can add these annotations in the [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resources so that advanced features in [Apache APISIX](https://apisix.apache.org) can be combined into [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress) resources.

> Note all keys and values of annotations are strings, so boolean value like `true` and `false` should be represented as `"true"` and `"false"`.

CORS Support
------------

In order to enable [CORS](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing), the annotation `k8s.apisix.apache.org/enable-cors` should be set to `"true"`, also, there are some other annotations to customize the cors behavior.

* `k8s.apisix.apache.org/cors-allow-origin`

This annotation controls which origins will be allowed, multiple origins join together with `,`, for instance: `https://foo.com,http://bar.com:8080`

Default value is `"*"`, which means all origins are allowed.

* `k8s.apisix.apache.org/cors-allow-headers`

This annotation controls which headers are accepted, multiple headers join together with `,`.

Default is `"*"`, which means all headers are accepted.

* `k8s.apisix.apache.org/cors-allow-methods`

This annotation controls which methods are accepted, multiple methods join together with `,`.

Default is `"*"`, which means all HTTP methods are accepted.

Allowlist Source Range
-----------------------

You can specify the allowed client IP addresses or nets by the annotation `k8s.apisix.apache.org/allowlist-source-range`, multiple IP adddresses or nets join together with `,`,
for instance, `k8s.apisix.apache.org/allowlist-source-range: 10.0.5.0/16,127.0.0.1,192.168.3.98`. Default value is *empty*, which means the sources are unlimited.
