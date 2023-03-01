---
title: traffic-split
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

This guide describes how to use the traffic-split plugin in APISIX Ingress.

The [traffic-split](https://apisix.apache.org/docs/apisix/plugins/traffic-split) plugin can be used to dynamically route part of traffic to various upstream services.

:::info IMPORTANT

If multiple services are declared in ApisixRoute `http[].backends`, it cannot be used with traffic-split.

**Correct example:**

```yaml

backends:
- serviceName: httpbin1
  servicePort: 80 
plugins:
- name: traffic-split # It works.
  enable: true

```

**Invalid examples:**

```yaml
backends:
- serviceName: httpbin1
  servicePort: 80 
- serviceName: httpbin2
  servicePort: 80
plugins:
- name: traffic-split # Not working, this plugin is invalid and cannot be used with multiple backend.
  enable: true
```

:::

## Example-usage

:::info IMPORTANT

The traffic-split plugin will directly use Kubernetes DNS, which will not automatically resolve services.

Because there is namespace isolation between services, and use the form of `${service}.${namespace}` in the `upstream.nodes`.

:::

### Deploy hecho1 and hecho2 service

We use [hashicorp/http-echo](https://hub.docker.com/r/hashicorp/http-echo) as the service image, See its overview page for details.

Deploy it to the default namespace:

```bash
# deployment hecho1
kubectl run hecho1 --image hashicorp/http-echo --port 5678 -- -text "http-echo v1"
kubectl expose pod hecho1 --port 5678

# deployment hecho2
kubectl run hecho2 --image hashicorp/http-echo --port 5678 -- -text "http-echo v2"
kubectl expose pod hecho2 --port 5678
```

### Route the traffic

Routes to different services according to the http header `subset`.

* If HTTP header `subset: v1` routing to hecho1.
* If HTTP header `subset: v2` routing to hecho2.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
    - name: rule1
      match:
        paths:
          - /hecho
      backends:
        - serviceName: hecho1
          servicePort: 5678
      plugins:
        - name: traffic-split
          enable: true
          config:
            rules:
              - match:
                  - vars:
                      - - http_subset
                        - ==
                        - v1
                weighted_upstreams:
                  - upstream:
                      name: v1
                      type: roundrobin
                      nodes:
                        hecho1.default:5678: 1
              - match:
                  - vars:
                      - - http_subset
                        - ==
                        - v2
                weighted_upstreams:
                  - upstream:
                      name: v2
                      type: roundrobin
                      nodes:
                        hecho2.default:5678: 1
```

Use header `sebset: v1` to access:

```shell
kubectl run -it --rm curl --restart=Never --image=curlimages/curl -- \
  curl http://apisix-gateway.ingress-apisix/hecho -H 'subset: v1' -i
```

It should output:

```shell
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Content-Length: 11
Connection: keep-alive
X-App-Name: http-echo
X-App-Version: 0.2.3
Date: Tue, 28 Feb 2023 08:54:31 GMT
Server: APISIX/3.1.0

http-echo v1
```

Use header `sebset: v2` to access:

```shell
kubectl run -it --rm curl --restart=Never --image=curlimages/curl -- \
  curl http://apisix-gateway.ingress-apisix/hecho -H 'subset: v2' -i
```

It should output:

```shell
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Content-Length: 11
Connection: keep-alive
X-App-Name: http-echo
X-App-Version: 0.2.3
Date: Tue, 28 Feb 2023 08:55:04 GMT
Server: APISIX/3.1.0

http-echo v2
```
