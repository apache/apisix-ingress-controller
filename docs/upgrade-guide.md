# APISIX Ingress Controller Upgrade Guide

## Upgrading from 1.x.x to 2.0.0: Key Changes and Considerations

This document outlines the major updates, configuration compatibility changes, API behavior differences, and critical considerations when upgrading the APISIX Ingress Controller from version 1.x.x to 2.0.0. Please read carefully and assess the impact on your existing system before proceeding with the upgrade.

### APISIX Version Dependency (Data Plane)

The `apisix-standalone` mode is supported only with **APISIX 3.13.0**. When using this mode, it is mandatory to upgrade the data plane APISIX instance along with the Ingress Controller.

### Architecture Changes

#### Architecture in 1.x.x

There were two main deployment architectures in 1.x.x:

| Mode           | Description                                                                            | Issue                                                                          |
| -------------- | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Admin API Mode | Runs a separate etcd instance, with APISIX Admin API managing data plane configuration | Complex to deploy; high maintenance overhead for etcd                          |
| Mock-ETCD Mode | APISIX and the Ingress Controller are deployed in the same Pod, mocking etcd endpoints | Stateless Ingress cannot persist revision info; may lead to data inconsistency |

#### Architecture in 2.0.0

![upgrade to 2.0.0 architecture](./assets/images/upgrade-to-architecture.png)

##### Mock-ETCD Mode Deprecated

The mock-etcd architecture is no longer supported. This mode introduced significant reliability issues: stateless ingress controllers could not persist revision metadata, leading to memory pollution in the data plane and data inconsistencies.

The following configuration block has been removed:

```yaml
etcdserver:
  enabled: false
  listen_address: ":12379"
  prefix: /apisix
  ssl_key_encrypt_salt: edd1c9f0985e76a2
```

##### Controller-Only Configuration Source

In 2.0.0, all data plane configurations must originate from the Ingress Controller. Configurations via Admin API or any external methods are no longer supported and will be ignored or may cause errors.

### Ingress Configuration Changes

#### Configuration Path Changes

| Old Path                 | New Path             |
| ------------------------ | -------------------- |
| `kubernetes.election_id` | `leader_election_id` |

#### Removed Configuration Fields

| Configuration Path   | Description                              |
| -------------------- | ---------------------------------------- |
| `kubernetes.*`       | Multi-namespace control / sync interval  |
| `plugin_metadata_cm` | Plugin metadata ConfigMap                |
| `log_rotation_*`     | Log rotation settings                    |
| `apisix.*`           | Static Admin API configuration           |
| `etcdserver.*`       | Configuration for mock-etcd (deprecated) |

#### Example: Legacy Configuration Removed in 2.0.0

```yaml
apisix:
  admin_api_version: v3
  default_cluster_base_url: "http://127.0.0.1:9180/apisix/admin"
  default_cluster_admin_key: ""
  default_cluster_name: "default"
```

#### New Configuration via `GatewayProxy` CRD

From version 2.0.0, the data plane must be connected via the `GatewayProxy` CRD:

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "default"
    scope: "Namespace"
---
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
  namespace: default
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - https://127.0.0.1:9180
      auth:
        type: AdminKey
        adminKey:
          value: ""
```

### API Changes

#### `ApisixUpstream`

Due to current limitations in the ADC (API Definition Controller) component, the following fields are not yet supported:

* `spec.discovery`: Service Discovery
* `spec.healthCheck`: Health Checking

More details: [ADC Backend Differences](https://github.com/api7/adc/blob/2449ca81e3c61169f8c1e59efb4c1173a766bce2/libs/backend-apisix-standalone/README.md#differences-in-upstream)

#### Limited Support for Ingress Annotations

Ingress annotations used in version 1.x.x are not fully supported in 2.0.0. If your existing setup relies on any of the following annotations, validate compatibility or consider delaying the upgrade.

| Ingress Annotations                                    |
| ------------------------------------------------------ |
| `k8s.apisix.apache.org/use-regex`                      |
| `k8s.apisix.apache.org/enable-websocket`               |
| `k8s.apisix.apache.org/plugin-config-name`             |
| `k8s.apisix.apache.org/upstream-scheme`                |
| `k8s.apisix.apache.org/upstream-retries`               |
| `k8s.apisix.apache.org/upstream-connect-timeout`       |
| `k8s.apisix.apache.org/upstream-read-timeout`          |
| `k8s.apisix.apache.org/upstream-send-timeout`          |
| `k8s.apisix.apache.org/enable-cors`                    |
| `k8s.apisix.apache.org/cors-allow-origin`              |
| `k8s.apisix.apache.org/cors-allow-headers`             |
| `k8s.apisix.apache.org/cors-allow-methods`             |
| `k8s.apisix.apache.org/enable-csrf`                    |
| `k8s.apisix.apache.org/csrf-key`                       |
| `k8s.apisix.apache.org/http-to-https`                  |
| `k8s.apisix.apache.org/http-redirect`                  |
| `k8s.apisix.apache.org/http-redirect-code`             |
| `k8s.apisix.apache.org/rewrite-target`                 |
| `k8s.apisix.apache.org/rewrite-target-regex`           |
| `k8s.apisix.apache.org/rewrite-target-regex-template`  |
| `k8s.apisix.apache.org/enable-response-rewrite`        |
| `k8s.apisix.apache.org/response-rewrite-status-code`   |
| `k8s.apisix.apache.org/response-rewrite-body`          |
| `k8s.apisix.apache.org/response-rewrite-body-base64`   |
| `k8s.apisix.apache.org/response-rewrite-add-header`    |
| `k8s.apisix.apache.org/response-rewrite-set-header`    |
| `k8s.apisix.apache.org/response-rewrite-remove-header` |
| `k8s.apisix.apache.org/auth-uri`                       |
| `k8s.apisix.apache.org/auth-ssl-verify`                |
| `k8s.apisix.apache.org/auth-request-headers`           |
| `k8s.apisix.apache.org/auth-upstream-headers`          |
| `k8s.apisix.apache.org/auth-client-headers`            |
| `k8s.apisix.apache.org/allowlist-source-range`         |
| `k8s.apisix.apache.org/blocklist-source-range`         |
| `k8s.apisix.apache.org/http-allow-methods`             |
| `k8s.apisix.apache.org/http-block-methods`             |
| `k8s.apisix.apache.org/auth-type`                      |
| `k8s.apisix.apache.org/svc-namespace`                  |

### Summary

| Category         | Description                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------- |
| Architecture     | The `mock-etcd` component has been removed. Configuration is now centralized through the Controller. |
| Configuration    | Static configuration fields have been removed. Use `GatewayProxy` CRD to configure the data plane.   |
| Data Plane       | Requires APISIX version 3.13.0 running in `standalone` mode.                                         |
| API              | Some fields in `Ingress Annotations` and `ApisixUpstream` are not yet supported.                     |
| Upgrade Strategy | Blue-green deployment or canary release is recommended before full switchover.                       |
