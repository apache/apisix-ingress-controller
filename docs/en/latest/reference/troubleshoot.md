---
title: Configuration Troubleshooting
slug: /reference/apisix-ingress-controller/configuration-troubleshoot
description: Learn how to inspect and troubleshoot configuration translation and synchronization in APISIX Ingress Controller.
---

Troubleshooting is required if the applied behavior does not match expectations, such as routes not being created correctly, plugins not being applied, or services failing to route traffic.

When you apply a Kubernetes resource—whether a Gateway API, Ingress, or APISIX CRD—the Ingress Controller translates it into ADC YAML, which is then applied to the gateway.

This document explains how to inspect the translated ADC configurations in memory and check the configurations actually applied to the gateway.

## Inspect Translated ADC Configurations

APISIX Ingress Controller provides a browser-accessible debug API that displays the translated ADC configurations, derived from the last applied Gateway API, Ingress, and APISIX CRD resources, in JSON format. It helps inspect the __in-memory state before the configurations are synchronized with the gateway__.

To use the debug API, configure these values in the ingress controller's [configuration file](./configuration-file.md):

```yaml title="config.yaml"
enable_server: true             # Enable the debug API server
server_addr: "127.0.0.1:9092"   # Server address
```

These values are not yet available in the Helm chart. To apply the changes, modify the ConfigMap and restart the controller Deployment.

Once the debug API is enabled, you can access it by forwarding the controller pod’s port to your local machine:

```shell
kubectl port-forward pod/<your-apisix-ingress-controller-pod-name> 9092:9092 &
```

You can now access the debug API in browser at `127.0.0.1:9092/debug` and inspect the translated resources by resource type, such as routes and services.

## Inspect Synchronized Gateway Configurations

To inspect the configurations synchronized to the gateway, you can use the Admin API.

First, forward the Admin API's service port to your local machine:

```shell
kubectl port-forward service/apisix-admin 9180:9180 &
```

If you are using APISIX in standalone mode, you can send a request to `/apisix/admin/configs` to view all configurations synchronized to the gateway:

```shell
curl "http://127.0.0.1:9180/apisix/admin/configs" -H "X-API-KEY: ${ADMIN_API_KEY}"
```

If you are using APISIX with etcd, you can send a request to `/apisix/admin/<resource>` to view the synchronized configurations of specific resources. For instance, to view the route configuration:

```shell
curl "http://127.0.0.1:9180/apisix/admin/routes" -H "X-API-KEY: ${ADMIN_API_KEY}"
```

For reference, see [Admin API](https://apisix.apache.org/docs/apisix/admin-api/).
