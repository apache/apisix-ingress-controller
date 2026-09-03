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

## Check Data Plane Instance Availability

When running APISIX in standalone mode with more than one instance, a route can be reachable through some instances but not others, for example if one instance is unreachable or rejects the synchronized configuration. This is not reflected on the affected route's own status, since the route itself was valid; it shows up on the `GatewayProxy` that addresses those instances.

Check the `DataPlaneAvailable` condition:

```shell
kubectl get gatewayproxy <gateway-proxy-name> -o yaml
```

```yaml
status:
  conditions:
  - type: DataPlaneAvailable
    status: "False"
    reason: DataPlaneInstanceUnavailable
    message: "1/3 gateway instance(s) failed to apply the last sync: http://apisix-2:9180: connection refused"
```

For the history of which specific instance failed and when, check the GatewayProxy's events:

```shell
kubectl describe gatewayproxy <gateway-proxy-name>
```

```
Events:
  Type     Reason                        Age   From             Message
  ----     ------                        ----  ----             -------
  Warning  DataPlaneInstanceUnavailable  8s    apisix-provider  http://apisix-2:9180: connection refused
```

Each unreachable or rejecting instance is reported as its own `Warning` event, so instances failing for different reasons, or at different times, don't get folded into one message.

## Gateway API Routes Return 404

Gateway API HTTPRoute or GRPCRoute resources may return `404` when the Gateway listener ports do not match the ports that APISIX actually listens on.

With [`listener_port_match_mode`](configuration-file.md) set to `"auto"` or `"explicit"`, the Ingress Controller injects a `server_port` route variable from the matched Gateway listener ports. APISIX evaluates `server_port` against the port it accepted the connection on, such as `9080` or `9443`. If the Gateway listener declares `80` or `443` but the gateway Service maps those ports to `9080` or `9443`, no route matches.

This is why `listener_port_match_mode` defaults to `"off"`. If you have enabled it, use one of the following approaches:

- Set [`listener_port_match_mode`](configuration-file.md) back to `"off"` to disable `server_port` route-var injection.
- Configure APISIX to listen on the same ports declared in the Gateway listeners.
