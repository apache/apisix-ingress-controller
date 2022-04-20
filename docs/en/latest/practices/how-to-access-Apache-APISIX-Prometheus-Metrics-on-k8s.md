---
title: How to access Apache APISIX Prometheus metrics on k8s
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

Prometheus is a leading open-source project focused on metrics and alerting that has changed the way the world does monitoring and observability. And Apache APISIX has enhanced its support for Prometheus Metrics in recent releases, adding a new feature for use in conjunction with the `public-api` plugin. This article will introduce how to configure `public-api` to protect Prometheus to collect Apache APISIX 's Metrics data. For more information, see Prometheus's official [website](https://prometheus.io/).

## Initial Knowledge about `public-api`

When users develop custom plugins in Apache APISIX, they can define some APIs (referred to as public API) for the plugins. The provided interface is for internal calls in practical application scenarios rather than being open on the public network for anyone to call.

Therefore, Apache APISIX has designed a `public-api` plugin. With this plugin, you can solve the public API's pain points. You can set a custom URI for the public API and configure any plugin. For more information, see the `public-api` Plugin's official [document](https://apisix.apache.org/docs/apisix/plugins/public-api/).

The primary role of the `public-api` plugin in this document is to protect the URI exposed by Prometheus.

**Note**: We should note that this feature is only available in APISIX version 2.13 and later.

## Begin to access Apache APISIX Prometheus Metrics

### Step 1: Install APISIX Ingress Controller

First, we deploy Apache APISIX, etcd, and APISIX Ingress Controller to a local Kubernetes cluster via Helm.

```sh
helm repo add apisix https://charts.apiseven.com
helm repo update
kubectl create namespace ingress-apisix
helm install apisix apisix/apisix --namespace ingress-apisix \
 --set ingress-controller.enabled=true
```

After installation, please wait until all services are up and running. And you can check specific status confirmation with the following command.

```sh
kubectl get all -n ingress-apisix
```

For more information, see [APISIX Ingress Controller the Hard Way](https://apisix.apache.org/docs/ingress-controller/practices/the-hard-way).

### Step 2: Enable Prometheus Plugin

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

### Step 3: Enable `public-api` Plugin

Let's make a basic routing setup, and please note that further configuration should be done based on your local backend service information. The primary solution concept is to use the `public-api` plugin to protect the routes exposed by Prometheus. For a more detailed configuration, you can refer to the [example](https://apisix.apache.org/zh/docs/apisix/plugins/public-api/#example) section of the `public-api` plugin.

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
    - serviceName: apisix-test-prometheus
      servicePort: 9180
    plugins:
    - name: public-api
      enable: true
```

### Step 4: Collect the Metrics

Now you can then get the indicator parameters by requesting command access.

```sh
kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9180/apisix/admin/routes -H 'X-API-Key: edd1c9f034335f136f87ad84b625c8f1'

kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9080/headers -H 'Host: test.prometheus.org'
```

## Conclusion

This article describes how to use the `public-api` plugin to protect Prometheus and monitor the Apache APISIX. Currently, only some basic configurations include. If you want to see some of the metrics displayed with Grafana, please refer to this [link](https://apisix.apache.org/zh/blog/2021/12/13/monitor-apisix-ingress-controller-with-prometheus/#). We will continue to polish and upgrade, add more metrics and integrate data surface APISIX metrics to improve your monitoring experience.

Of course, we welcome all interested parties to contribute to the [Apache APISIX Ingress Controller project](https://github.com/apache/apisix-ingress-controller) and look forward to working together to make the APISIX Ingress Controller more comprehensive.
