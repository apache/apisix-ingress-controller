---
title: Exporting Prometheus metrics from APISIX
keywords:
  - APISIX Ingress
  - Apache APISIX
  - Kubernetes Ingress
  - Prometheus
description: A tutorial on exporting Prometheus metrics from Apache APISIX Ingress.
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

This tutorial explains how you can export APISIX metrics in Prometheus format.

:::note

This tutorial requires APISIX version > 2.13.

:::

## Enable the prometheus Plugin

First, you have to enable the [prometheus](https://apisix.apache.org/docs/apisix/plugins/prometheus) Plugin. You can do this by adding to your `values.yaml` file while you install APISIX Ingress via Helm. You can also enable the [public-api](https://apisix.apache.org/docs/apisix/next/plugins/public-api) Plugin to expose these metrics.

A sample `values.yaml` file is shown below:

```yaml title="values.yaml"
gateway:
  type: NodePort

ingress-controller:
  enabled: true
  config:
    apisix:
      serviceNamespace: ingress-apisix

pluginAttrs:
  prometheus:
    enable_export_server: false

plugins:
  - api-breaker
  - authz-keycloak
  - basic-auth
  - batch-requests
  - consumer-restriction
  - cors
  - echo
  - fault-injection
  - file-logger
  - grpc-transcode
  - hmac-auth
  - http-logger
  - ip-restriction
  - ua-restriction
  - jwt-auth
  - kafka-logger
  - key-auth
  - limit-conn
  - limit-count
  - limit-req
  - node-status
  - openid-connect
  - authz-casbin
  - proxy-cache
  - proxy-mirror
  - proxy-rewrite
  - redirect
  - referer-restriction
  - request-id
  - request-validation
  - response-rewrite
  - serverless-post-function
  - serverless-pre-function
  - sls-logger
  - syslog
  - tcp-logger
  - udp-logger
  - uri-blocker
  - wolf-rbac
  - zipkin
  - traffic-split
  - gzip
  - real-ip
  - ext-plugin-pre-req
  - ext-plugin-post-req
  - prometheus # enable prometheus Plugin
  - public-api # enable public-api Plugin
```

You can install APISIX Ingress with Helm and pass this `values.yaml` file:

```shell
helm repo add apisix https://charts.apiseven.com
helm repo update
helm install apisix apisix/apisix -f values.yaml --create-namespace -n ingress-apisix
```

:::tip

APISIX also supports exporting HTTP request-related metrics like http_status, http_latency, and bandwidth. You can enable this by updating your configuration file as shown [here](https://apisix.apache.org/docs/apisix/next/plugins/prometheus/#specifying-metrics). 

:::

You should also configure APISIX Ingress with the prometheus Plugin by creating an [ApisixClusterConfig](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_cluster_config) resource as shown:

```yaml title="apisix-config.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
```

## Expose metrics with the public-api Plugin

You can use the `public-api` Plugin to expose the Prometheus metrics exported by APISIX. To do this, you can create a Route and enable the Plugin on it as shown below:

```yaml title="public-api.yaml"
apiVersion: apisix.apache.org/v2
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
    # replace this with your backend service
    - serviceName: apisix-admin
      servicePort: 9180
    plugins:
    - name: public-api
      enable: true
```

This will export the metrics to the `/apisix/prometheus/metrics` path.

## Check the exported metrics

You can configure Prometheus to pull APISIX's metrics from the `/apisix/prometheus/metrics` path. For testing, we will expose this path and check the exported metrics:

```sh
# forward to 127.0.0.1:9080
kubectl port-forward service/apisix-gateway 9080:80 -n ingress-apisix
```

```sh
curl http://127.0.0.1:9080/apisix/prometheus/metrics -H 'Host: test.prometheus.org'
```

This will show the metrics exported by APISIX:

```bash title="output"
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
