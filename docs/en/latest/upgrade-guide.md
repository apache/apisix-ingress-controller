---
title: Upgrade Guide
keywords:
  - APISIX Ingress
  - Apache APISIX
  - Kubernetes Ingress
  - Gateway API
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

## GatewayProxy control-plane TLS verification

TLS certificate verification is enabled by default for HTTPS control-plane
connections. Kubernetes CRD defaulting also applies to existing `GatewayProxy`
objects when they are read after the CRD is upgraded. As a result, an existing
object that omits `spec.provider.controlPlane.tlsVerify` will begin verifying
the control plane's certificate after upgrading to a release that contains
this change.

Before upgrading, check every `GatewayProxy` that uses an HTTPS control-plane
endpoint:

- No change is required when the certificate is trusted by the system trust
  store.
- For a self-signed certificate or private CA, configure the PEM-encoded CA
  certificate or bundle in `spec.provider.controlPlane.caCert.value`.
- Use `spec.provider.controlPlane.tlsVerify: false` only as a temporary
  development workaround. It disables certificate verification and exposes
  the AdminKey to man-in-the-middle attacks.

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

![upgrade to 2.0.0 architecture](../../assets/images/upgrade-to-architecture.png)

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

Starting with APISIX Ingress Controller 2.0.0, the controller is the single source of truth. Manual Admin API changes will be overwritten on the next full sync. The prior approach, which allowed controller-managed and manually added configurations to coexist, was incorrect and is now deprecated.

#### APISIX With Etcd (Admin API) synchronization performance

In APISIX Ingress Controller 2.0.0, ADC performs scheduled resource synchronization by comparing resources against the admin API response.

Because the Admin API fills in default values, the submitted content may differ from the returned result. This breaks the diff, triggering full updates to data plane resources, causing cache invalidation and significant performance impact.

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

#### Namespace Selector

`kubernetes.namespace_selector` is replaced by the top-level `namespace_selector`. Entries written for 1.x keep their meaning: every entry must match, and the values given for the same key are ORed. Each entry also accepts the full Kubernetes label selector syntax, such as `env in (prod,staging)` or `!legacy`. The command line flag `--namespace-selector` is not available, set the option in the configuration file.

It behaves differently from 1.x in the following ways:

- When a namespace stops matching, 2.x removes the configuration of its resources from the data plane, while 1.x left the synced routes in place. Before upgrading, check for namespaces that were unlabeled in 1.x but still have routes in service, since those routes disappear after the upgrade.
- Only Ingress and `apisix.apache.org/v2` resources are filtered. Gateway API resources, which 1.x also filtered, are not; use the `allowedRoutes` of the Gateway listeners to limit their namespaces.

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
    namespace: "ingress-apisix"
    scope: "Namespace"
---
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  namespace: ingress-apisix
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - http://127.0.0.1:9180
      auth:
        type: AdminKey
        adminKey:
          value: ""
```

### API Changes

#### `ApisixClusterConfig`

The `ApisixClusterConfig` CRD has been removed in 2.0.0. global rules and configurations should now be managed through the `ApisixGlobalRule` CRDs.

#### `ApisixConsumer` - `hmac-auth`

In apisix >= 3.11, most of the hmac-auth related configuration has been deprecated from consumer and moved to service/route level. The name of a `required` field has also been changed from `access_key` to `key_id`. If you have ApisixConsumer configuration with hmac-auth plugin compatible with <3.11, they will not be compatible with newer versions of APISIX. Since all 3+ versions of apisix are supported by ingress controller, if you dont upgrade APISIX, you don't need to change your ApisixConsumer configuration. But when using >3.11, pass all configurations other than `key_id`(previously `access_key`) via PluginConfig or ApisixRoute.

#### Ingress

##### API Version Support

Currently supports networking.k8s.io/v1 only. Support for other Ingress API versions (networking.k8s.io/v1beta1 and extensions/v1beta1) is not yet available in 2.0.0.

### Summary

| Category         | Description                                                                                                                                  |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Architecture     | The `mock-etcd` component has been removed. Configuration is now centralized through the Controller.                                         |
| Configuration    | Static configuration fields have been removed. Use `GatewayProxy` CRD to configure the data plane.                                           |
| Data Plane       | The Admin API configuration method is still supported. Support for the Standalone API-driven mode was introduced in APISIX 3.13.0 and later. |
| API              | Removed `ApisixClusterConfig`                                                                                                                |
| Upgrade Strategy | Blue-green deployment or canary release is recommended before full switchover.                                                               |
