---
title: How to access Apache APISIX Prometheus metrics on Kubernetes
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

Observability (monitoring functionality) has always played an essential role in system maintenance. A sound monitoring system can help engineers quickly understand the status of services running in production environments and locate problems or give early warning of anomalies when they occur.

*Prometheus* is a leading open-source project focused on metrics and alerting that has changed the way the world does monitoring and observability. For more information, see *Prometheus*'s official [website](https://prometheus.io/).

## Begin to access Apache APISIX Prometheus Metrics

Before starting, please make sure that Apache APISIX (version >= 2.13)and APISIX Ingress controller are installed and working correctly. APISIX uses the `prometheus` plugin to expose metrics and integrate with prometheus but uses `public-api` plugin to enhance its security after version 2.13. For more information, see `public-api` plugin's official [document](https://apisix.apache.org/docs/apisix/plugins/public-api/).

### Step 1: Enable Prometheus Plugin

If you need to monitor Apache APISIX simultaneously, you can create the following ApisixClusterConfig resource.

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
```

### Step 2: Enable `public-api` Plugin

Let's make a basic routing setup, and please note that further configuration should be done based on your local backend service information. The primary solution concept is to use the `public-api` plugin to protect the routes exposed by *Prometheus*. For a more detailed configuration, you can refer to the [example](https://apisix.apache.org/docs/apisix/plugins/public-api/#example) section of the `public-api` plugin.

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: prometheus-route
spec:
  http:
  - name: public-api
    match:
      hosts:
      - test.prometheus.org
      paths:
      - /apisix/prometheus/metrics
    backends:
    ## Please notice that there must be your actual "serviceName" and "servicePort"
    - serviceName: apisix-admin
      servicePort: 9180
    plugins:
    - name: public-api
      enable: true
```

### Step 3: Collect the Metrics

Now you can then get the indicator parameters by requesting command access.

```sh
kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9180/apisix/admin/routes -H 'X-API-Key: edd1c9f034335f136f87ad84b625c8f1'

kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9091/headers -H 'Host: test.prometheus.org'
```

Then you will get the metrics you want.

```bash
chever@cloud-native-01:~/api7/cloud_native/tasks/doc_prometheus$ kubectl exec -it -n ingress-apisix apisix-7d6b8577b6-rqhq9 -- curl http://127.0.0.1:9091/apisix/prometheus/metrics
Defaulted container "apisix" out of: apisix, wait-etcd (init)
# HELP apisix_bandwidth Total bandwidth in bytes consumed per service in APISIX
# TYPE apisix_bandwidth counter
apisix_bandwidth{type="egress",route="",service="",consumer="",node=""} 1130
apisix_bandwidth{type="ingress",route="",service="",consumer="",node=""} 517
# HELP apisix_etcd_modify_indexes Etcd modify index for APISIX keys
# TYPE apisix_etcd_modify_indexes gauge
apisix_etcd_modify_indexes{key="consumers"} 0
apisix_etcd_modify_indexes{key="global_rules"} 13
apisix_etcd_modify_indexes{key="max_modify_index"} 13
apisix_etcd_modify_indexes{key="prev_index"} 13
apisix_etcd_modify_indexes{key="protos"} 0
apisix_etcd_modify_indexes{key="routes"} 0
apisix_etcd_modify_indexes{key="services"} 0
apisix_etcd_modify_indexes{key="ssls"} 0
apisix_etcd_modify_indexes{key="stream_routes"} 0
apisix_etcd_modify_indexes{key="upstreams"} 0
apisix_etcd_modify_indexes{key="x_etcd_index"} 13
# HELP apisix_etcd_reachable Config server etcd reachable from APISIX, 0 is unreachable
# TYPE apisix_etcd_reachable gauge
apisix_etcd_reachable 1
# HELP apisix_http_latency HTTP request latency in milliseconds per service in APISIX
# TYPE apisix_http_latency histogram
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="1"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="2"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="5"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="10"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="20"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="50"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="100"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="200"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="500"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="1000"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="2000"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="5000"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="10000"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="30000"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="60000"} 5
apisix_http_latency_bucket{type="apisix",route="",service="",consumer="",node="",le="+Inf"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="1"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="2"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="5"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="10"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="20"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="50"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="100"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="200"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="500"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="1000"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="2000"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="5000"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="10000"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="30000"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="60000"} 5
apisix_http_latency_bucket{type="request",route="",service="",consumer="",node="",le="+Inf"} 5
apisix_http_latency_count{type="apisix",route="",service="",consumer="",node=""} 5
apisix_http_latency_count{type="request",route="",service="",consumer="",node=""} 5
apisix_http_latency_sum{type="apisix",route="",service="",consumer="",node=""} 0
apisix_http_latency_sum{type="request",route="",service="",consumer="",node=""} 0
# HELP apisix_http_requests_total The total number of client requests since APISIX started
# TYPE apisix_http_requests_total gauge
apisix_http_requests_total 82
# HELP apisix_http_status HTTP status codes per service in APISIX
# TYPE apisix_http_status counter
apisix_http_status{code="404",route="",matched_uri="",matched_host="",service="",consumer="",node=""} 5
# HELP apisix_nginx_http_current_connections Number of HTTP connections
# TYPE apisix_nginx_http_current_connections gauge
apisix_nginx_http_current_connections{state="accepted"} 2346
apisix_nginx_http_current_connections{state="active"} 1
apisix_nginx_http_current_connections{state="handled"} 2346
apisix_nginx_http_current_connections{state="reading"} 0
apisix_nginx_http_current_connections{state="waiting"} 0
apisix_nginx_http_current_connections{state="writing"} 1
# HELP apisix_nginx_metric_errors_total Number of nginx-lua-prometheus errors
# TYPE apisix_nginx_metric_errors_total counter
apisix_nginx_metric_errors_total 0
# HELP apisix_node_info Info of APISIX node
# TYPE apisix_node_info gauge
apisix_node_info{hostname="apisix-7d6b8577b6-rqhq9"} 1
```

## Conclusion

This article describes how to use the `public-api` plugin to protect *Prometheus* and monitor the Apache APISIX. Currently, only some basic configurations include. We will continue to polish and upgrade, add more metrics and integrate data surface APISIX metrics to improve your monitoring experience.

Of course, we welcome all interested parties to contribute to the [Apache APISIX Ingress Controller project](https://github.com/apache/apisix-ingress-controller) and look forward to working together to make the APISIX Ingress Controller more comprehensive.
