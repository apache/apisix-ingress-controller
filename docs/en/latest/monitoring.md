---
title: Monitoring APISIX with Helm Chart
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

APISIX has detailed telemetry data. With helm chart, we can easily configure the monitoring system.

This tutorial will show how to achieve it.

## Install Prometheus and Grafana

To collect metrics and visualize them, we need to install Prometheus and Grafana first.

The APISIX helm chart we will deploy later also contains a `ServiceMonitor` resource, so we should ensure the cluster has its CRD installed. Installing Prometheus Operator will apply the required CRD.

Run the following command to install Prometheus Operator and Grafana:

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install -n monitoring prometheus prometheus-community/kube-prometheus-stack \
  --create-namespace \
  --set 'prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false' \
  --set ingress-controller.enabled=true
```

We set option `prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues` to false to force Prometheus watches all service monitors in cluster for test purpose.

The default Grafana username and password is `admin` and `prom-operator`.

## Install APISIX

We should enable service monitor to tell Prometheus collect metrics from APISIX.

Install APISIX via helm chart with `serviceMonitor.enabled=true` option:

```bash
helm repo add apisix https://charts.apiseven.com
helm repo update

helm install apisix apisix/apisix --create-namespace --set serviceMonitor.enabled=true \
  --namespace apisix
```

## Configure Grafana Dashboard


Import [APISIX Grafana dashboard](https://grafana.com/grafana/dashboards/11719-apache-apisix/) via dashboard ID `11719`.

The dashboard should be able to display some data, including total requests, handled connections, etc. Routing related panels such as bandwidth and latency will show "No data" because we haven't made any requests yet. Make some requests to make these panels work.

## Manual Configuration and Troubleshooting

If you already have an installation of APISIX and Prometheus, you can manually configure `ServiceMonitor` resource and the service that exposes APISIX metrics.

### Service Monitor and APISIX Service

The magic behind `serviceMonitor.enabled=true` helm chart option is `ServiceMonitor` resource. Its content is as follows.

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
...
spec:
  namespaceSelector:
    matchNames:
    - apisix
  selector:
    matchLabels:
      helm.sh/chart: apisix-1.1.1
      app.kubernetes.io/name: apisix
      app.kubernetes.io/instance: apisix
      app.kubernetes.io/version: "3.1.1"
      app.kubernetes.io/managed-by: Helm
      app.kubernetes.io/service: apisix-gateway
  endpoints:
  - scheme: http
    targetPort: prometheus
    path: /apisix/prometheus/metrics
    interval: 15s
```

The spec uses `namespaceSelector` and `selector` to match the `apisix-gateway` service. The former matches the namespace of APISIX we deployed, and the later matches the exact service `apisix-gateway`.

The field `endpoints` tells Prometheus where to collect the metrics. Note that the `targetPort` field points to the port of service with the same name. If your service doesn't have a port named `prometheus`, create one.

The helm chart exposes APISIX metrics in the `apisix-gateway` service by default. Change the selector to match your own service if needed.

### Prometheus Spec

If everything works fine, the `Status/Targets` page of Promethes Web UI will show the APISIX service monitor. If your don't see it, you should make sure Prometheus is watching the `ServiceMonitor` resource we created.

By default, the `Prometheus` resource created by the helm chart `kube-prometheus-stack` is as follows.

```yaml
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
...
spec:
  ...
  serviceMonitorNamespaceSelector: {}
  serviceMonitorSelector:
    matchLabels:
      release: prometheus
```

Thus, we pass the option `prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false` to clear the `serviceMonitorSelector` field. Configure this resource to fit your needs.
