---
title: Resource Definitions API Reference
slug: /reference/apisix-ingress-controller/crd-reference
description: Explore detailed reference documentation for the custom resource definitions (CRDs) supported by the APISIX Ingress Controller.
---

This document provides the API resource description for the APISIX Ingress Controller.

## Packages
- [apisix.apache.org/v1alpha1](#apisixapacheorgv1alpha1)


## apisix.apache.org/v1alpha1

Package v1alpha1 contains API Schema definitions for the apisix.apache.org v1alpha1 API group

- [BackendTrafficPolicy](#backendtrafficpolicy)
- [Consumer](#consumer)
- [GatewayProxy](#gatewayproxy)
- [HTTPRoutePolicy](#httproutepolicy)
- [PluginConfig](#pluginconfig)
### BackendTrafficPolicy




<!-- BackendTrafficPolicy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `BackendTrafficPolicy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[BackendTrafficPolicySpec](#backendtrafficpolicyspec)_ | BackendTrafficPolicySpec defines traffic handling policies applied to backend services, such as load balancing strategy, connection settings, and failover behavior. |



### Consumer




<!-- Consumer resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `Consumer`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ConsumerSpec](#consumerspec)_ | ConsumerSpec defines the configuration for a consumer, including consumer name, authentication credentials, and plugin settings. |



### GatewayProxy


GatewayProxy is the Schema for the gatewayproxies API.

<!-- GatewayProxy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `GatewayProxy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[GatewayProxySpec](#gatewayproxyspec)_ | GatewayProxySpec defines the desired state and configuration of a GatewayProxy, including networking settings, global plugins, and plugin metadata. |



### HTTPRoutePolicy


HTTPRoutePolicy is the Schema for the httproutepolicies API.

<!-- HTTPRoutePolicy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `HTTPRoutePolicy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[HTTPRoutePolicySpec](#httproutepolicyspec)_ | HTTPRoutePolicySpec defines the desired state and configuration of a HTTPRoutePolicy, including route priority and request matching conditions. |



### PluginConfig


PluginConfig is the Schema for the PluginConfigs API.

<!-- PluginConfig resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `PluginConfig`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[PluginConfigSpec](#pluginconfigspec)_ | PluginConfigSpec defines the desired state of a PluginConfig, in which plugins and their configurations are specified. |



### Types

In this section you will find types that the CRDs rely on.
#### AdminKeyAuth


AdminKeyAuth defines the admin key authentication configuration.



| Field | Description |
| --- | --- |
| `value` _string_ | Value sets the admin key value explicitly (not recommended for production). |
| `valueFrom` _[AdminKeyValueFrom](#adminkeyvaluefrom)_ | ValueFrom specifies the source of the admin key. |


_Appears in:_
- [ControlPlaneAuth](#controlplaneauth)

#### AdminKeyValueFrom


AdminKeyValueFrom defines the source of the admin key.



| Field | Description |
| --- | --- |
| `secretKeyRef` _[SecretKeySelector](#secretkeyselector)_ | SecretKeyRef references a key in a Secret. |


_Appears in:_
- [AdminKeyAuth](#adminkeyauth)

#### AuthType
_Base type:_ `string`

AuthType defines the type of authentication.





_Appears in:_
- [ControlPlaneAuth](#controlplaneauth)

#### BackendPolicyTargetReferenceWithSectionName
_Base type:_ `LocalPolicyTargetReferenceWithSectionName`





| Field | Description |
| --- | --- |
| `group` _[Group](#group)_ | Group is the group of the target resource. |
| `kind` _[Kind](#kind)_ | Kind is kind of the target resource. |
| `name` _[ObjectName](#objectname)_ | Name is the name of the target resource. |
| `sectionName` _[SectionName](#sectionname)_ | SectionName is the name of a section within the target resource. When unspecified, this targetRef targets the entire resource. In the following resources, SectionName is interpreted as the following:<br /><br /> * Gateway: Listener name * HTTPRoute: HTTPRouteRule name * Service: Port name<br /><br /> If a SectionName is specified, but does not exist on the targeted object, the Policy must fail to attach, and the policy implementation should record a `ResolvedRefs` or similar Condition in the Policy's status. |


_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

#### BackendTrafficPolicySpec






| Field | Description |
| --- | --- |
| `targetRefs` _[BackendPolicyTargetReferenceWithSectionName](#backendpolicytargetreferencewithsectionname) array_ | TargetRef identifies an API object to apply policy to. Currently, Backends (i.e. Service, ServiceImport, or any implementation-specific backendRef) are the only valid API target references. |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer represents the load balancer configuration for Kubernetes Service. The default strategy is round robin. |
| `scheme` _string_ | Scheme is the protocol used to communicate with the upstream. Default is `http`. Can be one of `http`, `https`, `grpc`, or `grpcs`. |
| `retries` _integer_ | Retries specify the number of times the gateway should retry sending requests when errors such as timeouts or 502 errors occur. |
| `timeout` _[Timeout](#timeout)_ | Timeout sets the read, send, and connect timeouts to the upstream. |
| `passHost` _string_ | PassHost configures how the host header should be determined when a request is forwarded to the upstream. Default is `pass`. Can be one of `pass`, `node` or `rewrite`. |
| `upstreamHost` _[Hostname](#hostname)_ | UpstreamHost specifies the host of the Upstream request. Used only if passHost is set to `rewrite`. |


_Appears in:_
- [BackendTrafficPolicy](#backendtrafficpolicy)

#### ConsumerSpec






| Field | Description |
| --- | --- |
| `gatewayRef` _[GatewayRef](#gatewayref)_ | GatewayRef specifies the gateway details. |
| `credentials` _[Credential](#credential) array_ | Credentials specifies the credential details of a consumer. |
| `plugins` _[Plugin](#plugin) array_ | Plugins define the plugins associated with a consumer. |


_Appears in:_
- [Consumer](#consumer)

#### ControlPlaneAuth


ControlPlaneAuth defines the authentication configuration for control plane.



| Field | Description |
| --- | --- |
| `type` _[AuthType](#authtype)_ | Type specifies the type of authentication. Can only be `AdminKey`. |
| `adminKey` _[AdminKeyAuth](#adminkeyauth)_ | AdminKey specifies the admin key authentication configuration. |


_Appears in:_
- [ControlPlaneProvider](#controlplaneprovider)

#### ControlPlaneProvider


ControlPlaneProvider defines the configuration for control plane provider.



| Field | Description |
| --- | --- |
| `endpoints` _string array_ | Endpoints specifies the list of control plane endpoints. |
| `tlsVerify` _boolean_ | TlsVerify specifies whether to verify the TLS certificate of the control plane. |
| `auth` _[ControlPlaneAuth](#controlplaneauth)_ | Auth specifies the authentication configurations. |


_Appears in:_
- [GatewayProxyProvider](#gatewayproxyprovider)

#### Credential






| Field | Description |
| --- | --- |
| `type` _string_ | Type specifies the type of authentication to configure credentials for. Can be one of `jwt-auth`, `basic-auth`, `key-auth`, or `hmac-auth`. |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)_ | Config specifies the credential details for authentication. |
| `secretRef` _[SecretReference](#secretreference)_ | SecretRef references to the Secret that contains the credentials. |
| `name` _string_ | Name is the name of the credential. |


_Appears in:_
- [ConsumerSpec](#consumerspec)

#### GatewayProxyPlugin


GatewayProxyPlugin contains plugin configurations.



| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the plugin. |
| `enabled` _boolean_ | Enabled defines whether the plugin is enabled. |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)_ | Config defines the plugin's configuration details. |


_Appears in:_
- [GatewayProxySpec](#gatewayproxyspec)

#### GatewayProxyProvider


GatewayProxyProvider defines the provider configuration for GatewayProxy.



| Field | Description |
| --- | --- |
| `type` _[ProviderType](#providertype)_ | Type specifies the type of provider. Can only be `ControlPlane`. |
| `controlPlane` _[ControlPlaneProvider](#controlplaneprovider)_ | ControlPlane specifies the configuration for control plane provider. |


_Appears in:_
- [GatewayProxySpec](#gatewayproxyspec)

#### GatewayProxySpec


GatewayProxySpec defines the desired state of GatewayProxy.



| Field | Description |
| --- | --- |
| `publishService` _string_ | PublishService specifies the LoadBalancer-type Service whose external address the controller uses to update the status of Ingress resources. |
| `statusAddress` _string array_ | StatusAddress specifies the external IP addresses that the controller uses to populate the status field of GatewayProxy or Ingress resources for developers to access. |
| `provider` _[GatewayProxyProvider](#gatewayproxyprovider)_ | Provider configures the provider details. |
| `plugins` _[GatewayProxyPlugin](#gatewayproxyplugin) array_ | Plugins configure global plugins. |
| `pluginMetadata` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io))_ | PluginMetadata configures common configurations shared by all plugin instances of the same name. |


_Appears in:_
- [GatewayProxy](#gatewayproxy)

#### GatewayRef






| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the gateway. |
| `kind` _string_ | Kind is the type of Kubernetes object. Default is `Gateway`. |
| `group` _string_ | Group is the API group the resource belongs to. Default is `gateway.networking.k8s.io`. |
| `namespace` _string_ | Namespace is namespace of the resource. |


_Appears in:_
- [ConsumerSpec](#consumerspec)

#### HTTPRoutePolicySpec


HTTPRoutePolicySpec defines the desired state of HTTPRoutePolicy.



| Field | Description |
| --- | --- |
| `targetRefs` _LocalPolicyTargetReferenceWithSectionName array_ | TargetRef identifies an API object (i.e. HTTPRoute, Ingress) to apply HTTPRoutePolicy to. |
| `priority` _integer_ | Priority sets the priority for route. A higher value sets a higher priority in route matching. |
| `vars` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io) array_ | Vars sets the request matching conditions. |


_Appears in:_
- [HTTPRoutePolicy](#httproutepolicy)

#### Hostname
_Base type:_ `string`







_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

#### LoadBalancer


LoadBalancer describes the load balancing parameters.



| Field | Description |
| --- | --- |
| `type` _string_ | Type specifies the load balancing algorithms. Default is `roundrobin`. Can be one of `roundrobin`, `chash`, `ewma`, or `least_conn`. |
| `hashOn` _string_ | HashOn specified the type of field used for hashing, required when Type is `chash`. Default is `vars`. Can be one of `vars`, `header`, `cookie`, `consumer`, or `vars_combinations`. |
| `key` _string_ | Key is used with HashOn, generally required when Type is `chash`. When HashOn is `header` or `cookie`, specifies the name of the header or cookie. When HashOn is `consumer`, key is not required, as the consumer name is used automatically. When HashOn is `vars` or `vars_combinations`, key refers to one or a combination of [built-in variables](/enterprise/reference/built-in-variables). |


_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

#### Plugin






| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the plugin. |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)_ | Config is plugin configuration details. |


_Appears in:_
- [ConsumerSpec](#consumerspec)
- [PluginConfigSpec](#pluginconfigspec)

#### PluginConfigSpec


PluginConfigSpec defines the desired state of PluginConfig.



| Field | Description |
| --- | --- |
| `plugins` _[Plugin](#plugin) array_ | Plugins are an array of plugins and their configurations to be applied. |


_Appears in:_
- [PluginConfig](#pluginconfig)



#### ProviderType
_Base type:_ `string`

ProviderType defines the type of provider.





_Appears in:_
- [GatewayProxyProvider](#gatewayproxyprovider)

#### SecretKeySelector


SecretKeySelector defines a reference to a specific key within a Secret.



| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the secret. |
| `key` _string_ | Key is the key in the secret to retrieve the secret from. |


_Appears in:_
- [AdminKeyValueFrom](#adminkeyvaluefrom)

#### SecretReference






| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the secret. |
| `namespace` _string_ | Namespace is the namespace of the secret. |


_Appears in:_
- [Credential](#credential)



#### Timeout






| Field | Description |
| --- | --- |
| `connect` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Connection timeout. Default is `60s`. |
| `send` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Send timeout. Default is `60s`. |
| `read` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Read timeout. Default is `60s`. |


_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)

