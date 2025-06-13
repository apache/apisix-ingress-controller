---
title: Custom Resource Definitions API Reference
slug: /reference/apisix-ingress-controller/crd-reference
description: Explore detailed reference documentation for the custom resource definitions (CRDs) supported by the APISIX Ingress Controller.
---

This document provides the API resource description the API7 Ingress Controller custom resource definitions (CRDs).

## Packages
- [apisix.apache.org/v1alpha1](#apisixapacheorgv1alpha1)
- [apisix.apache.org/v2](#apisixapacheorgv2)


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
| `sectionName` _[SectionName](#sectionname)_ | SectionName is the name of a section within the target resource. When unspecified, this targetRef targets the entire resource. In the following resources, SectionName is interpreted as the following:<br /><br /> • Gateway: Listener name<br /> • HTTPRoute: HTTPRouteRule name<br /> • Service: Port name<br /><br /> If a SectionName is specified, but does not exist on the targeted object, the Policy must fail to attach, and the policy implementation should record a `ResolvedRefs` or similar Condition in the Policy's status. |


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
| `service` _[ProviderService](#providerservice)_ |  |
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



#### ProviderService






| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `port` _integer_ |  |


_Appears in:_
- [ControlPlaneProvider](#controlplaneprovider)

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

#### Status






| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#condition-v1-meta) array_ |  |


_Appears in:_
- [ConsumerStatus](#consumerstatus)

#### Timeout






| Field | Description |
| --- | --- |
| `connect` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Connection timeout. Default is `60s`. |
| `send` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Send timeout. Default is `60s`. |
| `read` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Read timeout. Default is `60s`. |


_Appears in:_
- [BackendTrafficPolicySpec](#backendtrafficpolicyspec)


## apisix.apache.org/v2

Package v2 contains API Schema definitions for the apisix.apache.org v2 API group.

- [ApisixConsumer](#apisixconsumer)
- [ApisixGlobalRule](#apisixglobalrule)
- [ApisixPluginConfig](#apisixpluginconfig)
- [ApisixRoute](#apisixroute)
- [ApisixTls](#apisixtls)
- [ApisixUpstream](#apisixupstream)
### ApisixConsumer


ApisixConsumer is the Schema for the apisixconsumers API.

<!-- ApisixConsumer resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixConsumer`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixConsumerSpec](#apisixconsumerspec)_ |  |



### ApisixGlobalRule


ApisixGlobalRule is the Schema for the apisixglobalrules API.

<!-- ApisixGlobalRule resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixGlobalRule`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixGlobalRuleSpec](#apisixglobalrulespec)_ |  |



### ApisixPluginConfig


ApisixPluginConfig is the Schema for the apisixpluginconfigs API.

<!-- ApisixPluginConfig resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixPluginConfig`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixPluginConfigSpec](#apisixpluginconfigspec)_ |  |



### ApisixRoute


ApisixRoute is the Schema for the apisixroutes API.

<!-- ApisixRoute resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixRoute`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixRouteSpec](#apisixroutespec)_ |  |



### ApisixTls


ApisixTls is the Schema for the apisixtls API.

<!-- ApisixTls resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixTls`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixTlsSpec](#apisixtlsspec)_ |  |



### ApisixUpstream


ApisixUpstream is the Schema for the apisixupstreams API.

<!-- ApisixUpstream resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixUpstream`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixUpstreamSpec](#apisixupstreamspec)_ |  |



### Types

In this section you will find types that the CRDs rely on.
#### ActiveHealthCheck


ActiveHealthCheck defines the active kind of upstream health check.



| Field | Description |
| --- | --- |
| `type` _string_ |  |
| `timeout` _[Duration](#duration)_ |  |
| `concurrency` _integer_ |  |
| `host` _string_ |  |
| `port` _integer_ |  |
| `httpPath` _string_ |  |
| `strictTLS` _boolean_ |  |
| `requestHeaders` _string array_ |  |
| `healthy` _[ActiveHealthCheckHealthy](#activehealthcheckhealthy)_ |  |
| `unhealthy` _[ActiveHealthCheckUnhealthy](#activehealthcheckunhealthy)_ |  |


_Appears in:_
- [HealthCheck](#healthcheck)

#### ActiveHealthCheckHealthy


ActiveHealthCheckHealthy defines the conditions to judge whether
an upstream node is healthy with the active manner.



| Field | Description |
| --- | --- |
| `httpCodes` _integer array_ |  |
| `successes` _integer_ |  |
| `interval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ |  |


_Appears in:_
- [ActiveHealthCheck](#activehealthcheck)

#### ActiveHealthCheckUnhealthy


ActiveHealthCheckUnhealthy defines the conditions to judge whether
an upstream node is unhealthy with the active manager.



| Field | Description |
| --- | --- |
| `httpCodes` _integer array_ |  |
| `httpFailures` _integer_ |  |
| `tcpFailures` _integer_ |  |
| `timeout` _integer_ |  |
| `interval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ |  |


_Appears in:_
- [ActiveHealthCheck](#activehealthcheck)

#### ApisixConsumerAuthParameter






| Field | Description |
| --- | --- |
| `basicAuth` _[ApisixConsumerBasicAuth](#apisixconsumerbasicauth)_ |  |
| `keyAuth` _[ApisixConsumerKeyAuth](#apisixconsumerkeyauth)_ |  |
| `wolfRBAC` _[ApisixConsumerWolfRBAC](#apisixconsumerwolfrbac)_ |  |
| `jwtAuth` _[ApisixConsumerJwtAuth](#apisixconsumerjwtauth)_ |  |
| `hmacAuth` _[ApisixConsumerHMACAuth](#apisixconsumerhmacauth)_ |  |
| `ldapAuth` _[ApisixConsumerLDAPAuth](#apisixconsumerldapauth)_ |  |


_Appears in:_
- [ApisixConsumerSpec](#apisixconsumerspec)

#### ApisixConsumerBasicAuth


ApisixConsumerBasicAuth defines the configuration for basic auth.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ |  |
| `value` _[ApisixConsumerBasicAuthValue](#apisixconsumerbasicauthvalue)_ |  |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerBasicAuthValue


ApisixConsumerBasicAuthValue defines the in-place username and password configuration for basic auth.



| Field | Description |
| --- | --- |
| `username` _string_ |  |
| `password` _string_ |  |


_Appears in:_
- [ApisixConsumerBasicAuth](#apisixconsumerbasicauth)

#### ApisixConsumerHMACAuth


ApisixConsumerHMACAuth defines the configuration for the hmac auth.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ |  |
| `value` _[ApisixConsumerHMACAuthValue](#apisixconsumerhmacauthvalue)_ |  |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerHMACAuthValue


ApisixConsumerHMACAuthValue defines the in-place configuration for hmac auth.



| Field | Description |
| --- | --- |
| `access_key` _string_ |  |
| `secret_key` _string_ |  |
| `algorithm` _string_ |  |
| `clock_skew` _integer_ |  |
| `signed_headers` _string array_ |  |
| `keep_headers` _boolean_ |  |
| `encode_uri_params` _boolean_ |  |
| `validate_request_body` _boolean_ |  |
| `max_req_body` _integer_ |  |


_Appears in:_
- [ApisixConsumerHMACAuth](#apisixconsumerhmacauth)

#### ApisixConsumerJwtAuth


ApisixConsumerJwtAuth defines the configuration for the jwt auth.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ |  |
| `value` _[ApisixConsumerJwtAuthValue](#apisixconsumerjwtauthvalue)_ |  |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerJwtAuthValue


ApisixConsumerJwtAuthValue defines the in-place configuration for jwt auth.



| Field | Description |
| --- | --- |
| `key` _string_ |  |
| `secret` _string_ |  |
| `public_key` _string_ |  |
| `private_key` _string_ |  |
| `algorithm` _string_ |  |
| `exp` _integer_ |  |
| `base64_secret` _boolean_ |  |
| `lifetime_grace_period` _integer_ |  |


_Appears in:_
- [ApisixConsumerJwtAuth](#apisixconsumerjwtauth)

#### ApisixConsumerKeyAuth


ApisixConsumerKeyAuth defines the configuration for the key auth.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ |  |
| `value` _[ApisixConsumerKeyAuthValue](#apisixconsumerkeyauthvalue)_ |  |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerKeyAuthValue


ApisixConsumerKeyAuthValue defines the in-place configuration for basic auth.



| Field | Description |
| --- | --- |
| `key` _string_ |  |


_Appears in:_
- [ApisixConsumerKeyAuth](#apisixconsumerkeyauth)

#### ApisixConsumerLDAPAuth


ApisixConsumerLDAPAuth defines the configuration for the ldap auth.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ |  |
| `value` _[ApisixConsumerLDAPAuthValue](#apisixconsumerldapauthvalue)_ |  |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerLDAPAuthValue


ApisixConsumerLDAPAuthValue defines the in-place configuration for ldap auth.



| Field | Description |
| --- | --- |
| `user_dn` _string_ |  |


_Appears in:_
- [ApisixConsumerLDAPAuth](#apisixconsumerldapauth)

#### ApisixConsumerSpec


ApisixConsumerSpec defines the desired state of ApisixConsumer.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. controller implementations use this field to know whether they should be serving this ApisixConsumer resource, by a transitive connection (controller -> IngressClass -> ApisixConsumer resource). |
| `authParameter` _[ApisixConsumerAuthParameter](#apisixconsumerauthparameter)_ |  |


_Appears in:_
- [ApisixConsumer](#apisixconsumer)

#### ApisixConsumerWolfRBAC


ApisixConsumerWolfRBAC defines the configuration for the wolf-rbac auth.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ |  |
| `value` _[ApisixConsumerWolfRBACValue](#apisixconsumerwolfrbacvalue)_ |  |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerWolfRBACValue


ApisixConsumerWolfRBAC defines the in-place server and appid and header_prefix configuration for wolf-rbac auth.



| Field | Description |
| --- | --- |
| `server` _string_ |  |
| `appid` _string_ |  |
| `header_prefix` _string_ |  |


_Appears in:_
- [ApisixConsumerWolfRBAC](#apisixconsumerwolfrbac)

#### ApisixGlobalRuleSpec


ApisixGlobalRuleSpec defines the desired state of ApisixGlobalRule.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. The controller uses this field to decide whether the resource should be managed or not. |
| `plugins` _[ApisixRoutePlugin](#apisixrouteplugin) array_ | Plugins contains a list of ApisixRoutePlugin |


_Appears in:_
- [ApisixGlobalRule](#apisixglobalrule)

#### ApisixMutualTlsClientConfig


ApisixMutualTlsClientConfig describes the mutual TLS CA and verify depth



| Field | Description |
| --- | --- |
| `caSecret` _[ApisixSecret](#apisixsecret)_ |  |
| `depth` _integer_ |  |
| `skip_mtls_uri_regex` _string array_ |  |


_Appears in:_
- [ApisixTlsSpec](#apisixtlsspec)

#### ApisixPluginConfigSpec


ApisixPluginConfigSpec defines the desired state of ApisixPluginConfigSpec.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. The controller uses this field to decide whether the resource should be managed or not. |
| `plugins` _[ApisixRoutePlugin](#apisixrouteplugin) array_ | Plugins contain a list of ApisixRoutePlugin |


_Appears in:_
- [ApisixPluginConfig](#apisixpluginconfig)



#### ApisixRouteAuthentication


ApisixRouteAuthentication is the authentication-related
configuration in ApisixRoute.



| Field | Description |
| --- | --- |
| `enable` _boolean_ |  |
| `type` _string_ |  |
| `keyAuth` _[ApisixRouteAuthenticationKeyAuth](#apisixrouteauthenticationkeyauth)_ |  |
| `jwtAuth` _[ApisixRouteAuthenticationJwtAuth](#apisixrouteauthenticationjwtauth)_ |  |
| `ldapAuth` _[ApisixRouteAuthenticationLDAPAuth](#apisixrouteauthenticationldapauth)_ |  |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)

#### ApisixRouteAuthenticationJwtAuth


ApisixRouteAuthenticationJwtAuth is the jwt auth related
configuration in ApisixRouteAuthentication.



| Field | Description |
| --- | --- |
| `header` _string_ |  |
| `query` _string_ |  |
| `cookie` _string_ |  |


_Appears in:_
- [ApisixRouteAuthentication](#apisixrouteauthentication)

#### ApisixRouteAuthenticationKeyAuth


ApisixRouteAuthenticationKeyAuth is the keyAuth-related
configuration in ApisixRouteAuthentication.



| Field | Description |
| --- | --- |
| `header` _string_ |  |


_Appears in:_
- [ApisixRouteAuthentication](#apisixrouteauthentication)

#### ApisixRouteAuthenticationLDAPAuth


ApisixRouteAuthenticationLDAPAuth is the LDAP auth related
configuration in ApisixRouteAuthentication.



| Field | Description |
| --- | --- |
| `base_dn` _string_ |  |
| `ldap_uri` _string_ |  |
| `use_tls` _boolean_ |  |
| `uid` _string_ |  |


_Appears in:_
- [ApisixRouteAuthentication](#apisixrouteauthentication)

#### ApisixRouteHTTP


ApisixRouteHTTP represents a single route in for HTTP traffic.



| Field | Description |
| --- | --- |
| `name` _string_ | The rule name, cannot be empty. |
| `priority` _integer_ | Route priority, when multiple routes contains same URI path (for path matching), route with higher priority will take effect. |
| `timeout` _[UpstreamTimeout](#upstreamtimeout)_ |  |
| `match` _[ApisixRouteHTTPMatch](#apisixroutehttpmatch)_ |  |
| `backends` _[ApisixRouteHTTPBackend](#apisixroutehttpbackend) array_ | Backends represents potential backends to proxy after the route rule matched. When number of backends are more than one, traffic-split plugin in APISIX will be used to split traffic based on the backend weight. |
| `upstreams` _[ApisixRouteUpstreamReference](#apisixrouteupstreamreference) array_ | Upstreams refer to ApisixUpstream CRD |
| `websocket` _boolean_ |  |
| `plugin_config_name` _string_ |  |
| `plugin_config_namespace` _string_ | By default, PluginConfigNamespace will be the same as the namespace of ApisixRoute |
| `plugins` _[ApisixRoutePlugin](#apisixrouteplugin) array_ |  |
| `authentication` _[ApisixRouteAuthentication](#apisixrouteauthentication)_ |  |


_Appears in:_
- [ApisixRouteSpec](#apisixroutespec)

#### ApisixRouteHTTPBackend


ApisixRouteHTTPBackend represents an HTTP backend (a Kubernetes Service).



| Field | Description |
| --- | --- |
| `serviceName` _string_ | The name (short) of the service, note cross namespace is forbidden, so be sure the ApisixRoute and Service are in the same namespace. |
| `servicePort` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#intorstring-intstr-util)_ | The service port, could be the name or the port number. |
| `resolveGranularity` _string_ | The resolve granularity, can be "endpoints" or "service", when set to "endpoints", the pod ips will be used; other wise, the service ClusterIP or ExternalIP will be used, default is endpoints. |
| `weight` _integer_ | Weight of this backend. |
| `subset` _string_ | Subset specifies a subset for the target Service. The subset should be pre-defined in ApisixUpstream about this service. |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)

#### ApisixRouteHTTPMatch


ApisixRouteHTTPMatch represents the match condition for hitting this route.



| Field | Description |
| --- | --- |
| `paths` _string array_ | URI path predicates, at least one path should be configured, path could be exact or prefix, for prefix path, append "*" after it, for instance, "/foo*". |
| `methods` _string array_ | HTTP request method predicates. |
| `hosts` _string array_ | HTTP Host predicates, host can be a wildcard domain or an exact domain. For wildcard domain, only one generic level is allowed, for instance, "*.foo.com" is valid but "*.*.foo.com" is not. |
| `remoteAddrs` _string array_ | Remote address predicates, items can be valid IPv4 address or IPv6 address or CIDR. |
| `exprs` _[ApisixRouteHTTPMatchExpr](#apisixroutehttpmatchexpr) array_ | NginxVars represents generic match predicates, it uses Nginx variable systems, so any predicate like headers, querystring and etc can be leveraged here to match the route. For instance, it can be: nginxVars:   - subject: "$remote_addr"     op: in     value:       - "127.0.0.1"       - "10.0.5.11" |
| `filter_func` _string_ | Matches based on a user-defined filtering function. These functions can accept an input parameter `vars` which can be used to access the Nginx variables. |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)

#### ApisixRouteHTTPMatchExpr


ApisixRouteHTTPMatchExpr represents a binary route match expression .



| Field | Description |
| --- | --- |
| `subject` _[ApisixRouteHTTPMatchExprSubject](#apisixroutehttpmatchexprsubject)_ | Subject is the expression subject, it can be any string composed by literals and nginx vars. |
| `op` _string_ | Op is the operator. |
| `set` _string array_ | Set is an array type object of the expression. It should be used when the Op is "in" or "not_in"; |
| `value` _string_ | Value is the normal type object for the expression, it should be used when the Op is not "in" and "not_in". Set and Value are exclusive so only of them can be set in the same time. |


_Appears in:_
- [ApisixRouteHTTPMatch](#apisixroutehttpmatch)

#### ApisixRouteHTTPMatchExprSubject


ApisixRouteHTTPMatchExprSubject describes the route match expression subject.



| Field | Description |
| --- | --- |
| `scope` _string_ | The subject scope, can be: ScopeQuery, ScopeHeader, ScopePath when subject is ScopePath, Name field will be ignored. |
| `name` _string_ | The name of subject. |


_Appears in:_
- [ApisixRouteHTTPMatchExpr](#apisixroutehttpmatchexpr)

#### ApisixRoutePlugin


ApisixRoutePlugin represents an APISIX plugin.



| Field | Description |
| --- | --- |
| `name` _string_ | The plugin name. |
| `enable` _boolean_ | Whether this plugin is in use, default is true. |
| `config` _[ApisixRoutePluginConfig](#apisixroutepluginconfig)_ | Plugin configuration. |
| `secretRef` _string_ | Plugin configuration secretRef. |


_Appears in:_
- [ApisixGlobalRuleSpec](#apisixglobalrulespec)
- [ApisixPluginConfigSpec](#apisixpluginconfigspec)
- [ApisixRouteHTTP](#apisixroutehttp)
- [ApisixRouteStream](#apisixroutestream)

#### ApisixRoutePluginConfig
_Base type:_ `[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)`

ApisixRoutePluginConfig is the configuration for
any plugins.





_Appears in:_
- [ApisixRoutePlugin](#apisixrouteplugin)

#### ApisixRouteSpec


ApisixRouteSpec is the spec definition for ApisixRouteSpec.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ |  |
| `http` _[ApisixRouteHTTP](#apisixroutehttp) array_ |  |
| `stream` _[ApisixRouteStream](#apisixroutestream) array_ |  |


_Appears in:_
- [ApisixRoute](#apisixroute)

#### ApisixRouteStream


ApisixRouteStream is the configuration for level 4 route



| Field | Description |
| --- | --- |
| `name` _string_ | The rule name cannot be empty. |
| `protocol` _string_ |  |
| `match` _[ApisixRouteStreamMatch](#apisixroutestreammatch)_ |  |
| `backend` _[ApisixRouteStreamBackend](#apisixroutestreambackend)_ |  |
| `plugins` _[ApisixRoutePlugin](#apisixrouteplugin) array_ |  |


_Appears in:_
- [ApisixRouteSpec](#apisixroutespec)

#### ApisixRouteStreamBackend


ApisixRouteStreamBackend represents a TCP backend (a Kubernetes Service).



| Field | Description |
| --- | --- |
| `serviceName` _string_ | The name (short) of the service, note cross namespace is forbidden, so be sure the ApisixRoute and Service are in the same namespace. |
| `servicePort` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#intorstring-intstr-util)_ | The service port, could be the name or the port number. |
| `resolveGranularity` _string_ | The resolve granularity, can be "endpoints" or "service", when set to "endpoints", the pod ips will be used; other wise, the service ClusterIP or ExternalIP will be used, default is endpoints. |
| `subset` _string_ | Subset specifies a subset for the target Service. The subset should be pre-defined in ApisixUpstream about this service. |


_Appears in:_
- [ApisixRouteStream](#apisixroutestream)

#### ApisixRouteStreamMatch


ApisixRouteStreamMatch represents the match conditions of stream route.



| Field | Description |
| --- | --- |
| `ingressPort` _integer_ | IngressPort represents the port listening on the Ingress proxy server. It should be pre-defined as APISIX doesn't support dynamic listening. |
| `host` _string_ |  |


_Appears in:_
- [ApisixRouteStream](#apisixroutestream)

#### ApisixRouteUpstreamReference


ApisixRouteUpstreamReference contains a ApisixUpstream CRD reference



| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `weight` _integer_ |  |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)

#### ApisixSecret


ApisixSecret describes the Kubernetes Secret name and namespace.



| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `namespace` _string_ |  |


_Appears in:_
- [ApisixMutualTlsClientConfig](#apisixmutualtlsclientconfig)
- [ApisixTlsSpec](#apisixtlsspec)
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### ApisixStatus


ApisixStatus is the status report for Apisix ingress Resources



| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#condition-v1-meta) array_ |  |


_Appears in:_
- [ApisixPluginConfigStatus](#apisixpluginconfigstatus)

#### ApisixStatus


ApisixStatus is the status report for Apisix ingress Resources



| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#condition-v1-meta) array_ |  |


_Appears in:_
- [ApisixPluginConfigStatus](#apisixpluginconfigstatus)

#### ApisixStatus


ApisixStatus is the status report for Apisix ingress Resources



| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#condition-v1-meta) array_ |  |


_Appears in:_
- [ApisixPluginConfigStatus](#apisixpluginconfigstatus)

#### ApisixStatus


ApisixStatus is the status report for Apisix ingress Resources



| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#condition-v1-meta) array_ |  |


_Appears in:_
- [ApisixPluginConfigStatus](#apisixpluginconfigstatus)

#### ApisixStatus


ApisixStatus is the status report for Apisix ingress Resources



| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#condition-v1-meta) array_ |  |


_Appears in:_
- [ApisixPluginConfigStatus](#apisixpluginconfigstatus)

#### ApisixStatus


ApisixStatus is the status report for Apisix ingress Resources



| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#condition-v1-meta) array_ |  |


_Appears in:_
- [ApisixPluginConfigStatus](#apisixpluginconfigstatus)

#### ApisixTlsSpec


ApisixTlsSpec defines the desired state of ApisixTls.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. controller implementations use this field to know whether they should be serving this ApisixTls resource, by a transitive connection (controller -> IngressClass -> ApisixTls resource). |
| `hosts` _[HostType](#hosttype) array_ |  |
| `secret` _[ApisixSecret](#apisixsecret)_ |  |
| `client` _[ApisixMutualTlsClientConfig](#apisixmutualtlsclientconfig)_ |  |


_Appears in:_
- [ApisixTls](#apisixtls)

#### ApisixUpstreamConfig


ApisixUpstreamConfig contains rich features on APISIX Upstream, for instance
load balancer, health check, etc.



| Field | Description |
| --- | --- |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer represents the load balancer configuration for Kubernetes Service. The default strategy is round robin. |
| `scheme` _string_ | The scheme used to talk with the upstream. Now value can be http, grpc. |
| `retries` _integer_ | How many times that the proxy (Apache APISIX) should do when errors occur (error, timeout or bad http status codes like 500, 502). |
| `timeout` _[UpstreamTimeout](#upstreamtimeout)_ | Timeout settings for the read, send and connect to the upstream. |
| `healthCheck` _[HealthCheck](#healthcheck)_ | The health check configurations for the upstream. |
| `tlsSecret` _[ApisixSecret](#apisixsecret)_ | Set the client certificate when connecting to TLS upstream. |
| `subsets` _[ApisixUpstreamSubset](#apisixupstreamsubset) array_ | Subsets groups the service endpoints by their labels. Usually used to differentiate service versions. |
| `passHost` _string_ | Configures the host when the request is forwarded to the upstream. Can be one of pass, node or rewrite. |
| `upstreamHost` _string_ | Specifies the host of the Upstream request. This is only valid if the pass_host is set to rewrite |
| `discovery` _[Discovery](#discovery)_ | Discovery is used to configure service discovery for upstream. |


_Appears in:_
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### ApisixUpstreamExternalNode


ApisixUpstreamExternalNode is the external node conf



| Field | Description |
| --- | --- |
| `name` _string_ |  |
| `type` _[ApisixUpstreamExternalType](#apisixupstreamexternaltype)_ |  |
| `weight` _integer_ |  |
| `port` _integer_ | Port defines the port of the external node |


_Appears in:_
- [ApisixUpstreamSpec](#apisixupstreamspec)

#### ApisixUpstreamExternalType
_Base type:_ `string`

ApisixUpstreamExternalType is the external service type





_Appears in:_
- [ApisixUpstreamExternalNode](#apisixupstreamexternalnode)

#### ApisixUpstreamSpec


ApisixUpstreamSpec describes the specification of ApisixUpstream.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. controller implementations use this field to know whether they should be serving this ApisixUpstream resource, by a transitive connection (controller -> IngressClass -> ApisixUpstream resource). |
| `externalNodes` _[ApisixUpstreamExternalNode](#apisixupstreamexternalnode) array_ | ExternalNodes contains external nodes the Upstream should use If this field is set, the upstream will use these nodes directly without any further resolves |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer represents the load balancer configuration for Kubernetes Service. The default strategy is round robin. |
| `scheme` _string_ | The scheme used to talk with the upstream. Now value can be http, grpc. |
| `retries` _integer_ | How many times that the proxy (Apache APISIX) should do when errors occur (error, timeout or bad http status codes like 500, 502). |
| `timeout` _[UpstreamTimeout](#upstreamtimeout)_ | Timeout settings for the read, send and connect to the upstream. |
| `healthCheck` _[HealthCheck](#healthcheck)_ | The health check configurations for the upstream. |
| `tlsSecret` _[ApisixSecret](#apisixsecret)_ | Set the client certificate when connecting to TLS upstream. |
| `subsets` _[ApisixUpstreamSubset](#apisixupstreamsubset) array_ | Subsets groups the service endpoints by their labels. Usually used to differentiate service versions. |
| `passHost` _string_ | Configures the host when the request is forwarded to the upstream. Can be one of pass, node or rewrite. |
| `upstreamHost` _string_ | Specifies the host of the Upstream request. This is only valid if the pass_host is set to rewrite |
| `discovery` _[Discovery](#discovery)_ | Discovery is used to configure service discovery for upstream. |
| `portLevelSettings` _[PortLevelSettings](#portlevelsettings) array_ |  |


_Appears in:_
- [ApisixUpstream](#apisixupstream)

#### ApisixUpstreamSubset


ApisixUpstreamSubset defines a single endpoints group of one Service.



| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of subset. |
| `labels` _object (keys:string, values:string)_ | Labels is the label set of this subset. |


_Appears in:_
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### Discovery


Discovery defines Service discovery related configuration.



| Field | Description |
| --- | --- |
| `serviceName` _string_ |  |
| `type` _string_ |  |
| `args` _object (keys:string, values:string)_ |  |


_Appears in:_
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### HealthCheck


HealthCheck describes the upstream health check parameters.



| Field | Description |
| --- | --- |
| `active` _[ActiveHealthCheck](#activehealthcheck)_ |  |
| `passive` _[PassiveHealthCheck](#passivehealthcheck)_ |  |


_Appears in:_
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### HostType
_Base type:_ `string`







_Appears in:_
- [ApisixTlsSpec](#apisixtlsspec)

#### LoadBalancer


LoadBalancer describes the load balancing parameters.



| Field | Description |
| --- | --- |
| `type` _string_ |  |
| `hashOn` _string_ | The HashOn and Key fields are required when Type is "chash". HashOn represents the key fetching scope. |
| `key` _string_ | Key represents the hash key. |


_Appears in:_
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### PassiveHealthCheck


PassiveHealthCheck defines the conditions to judge whether
an upstream node is healthy with the passive manager.



| Field | Description |
| --- | --- |
| `type` _string_ |  |
| `healthy` _[PassiveHealthCheckHealthy](#passivehealthcheckhealthy)_ |  |
| `unhealthy` _[PassiveHealthCheckUnhealthy](#passivehealthcheckunhealthy)_ |  |


_Appears in:_
- [HealthCheck](#healthcheck)

#### PassiveHealthCheckHealthy


PassiveHealthCheckHealthy defines the conditions to judge whether
an upstream node is healthy with the passive manner.



| Field | Description |
| --- | --- |
| `httpCodes` _integer array_ |  |
| `successes` _integer_ |  |


_Appears in:_
- [ActiveHealthCheckHealthy](#activehealthcheckhealthy)
- [PassiveHealthCheck](#passivehealthcheck)

#### PassiveHealthCheckUnhealthy


PassiveHealthCheckUnhealthy defines the conditions to judge whether
an upstream node is unhealthy with the passive manager.



| Field | Description |
| --- | --- |
| `httpCodes` _integer array_ |  |
| `httpFailures` _integer_ |  |
| `tcpFailures` _integer_ |  |
| `timeout` _integer_ |  |


_Appears in:_
- [ActiveHealthCheckUnhealthy](#activehealthcheckunhealthy)
- [PassiveHealthCheck](#passivehealthcheck)

#### PortLevelSettings


PortLevelSettings configures the ApisixUpstreamConfig for each individual port. It inherits
configurations from the outer level (the whole Kubernetes Service) and overrides some of
them if they are set on the port level.



| Field | Description |
| --- | --- |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer represents the load balancer configuration for Kubernetes Service. The default strategy is round robin. |
| `scheme` _string_ | The scheme used to talk with the upstream. Now value can be http, grpc. |
| `retries` _integer_ | How many times that the proxy (Apache APISIX) should do when errors occur (error, timeout or bad http status codes like 500, 502). |
| `timeout` _[UpstreamTimeout](#upstreamtimeout)_ | Timeout settings for the read, send and connect to the upstream. |
| `healthCheck` _[HealthCheck](#healthcheck)_ | The health check configurations for the upstream. |
| `tlsSecret` _[ApisixSecret](#apisixsecret)_ | Set the client certificate when connecting to TLS upstream. |
| `subsets` _[ApisixUpstreamSubset](#apisixupstreamsubset) array_ | Subsets groups the service endpoints by their labels. Usually used to differentiate service versions. |
| `passHost` _string_ | Configures the host when the request is forwarded to the upstream. Can be one of pass, node or rewrite. |
| `upstreamHost` _string_ | Specifies the host of the Upstream request. This is only valid if the pass_host is set to rewrite |
| `discovery` _[Discovery](#discovery)_ | Discovery is used to configure service discovery for upstream. |
| `port` _integer_ | Port is a Kubernetes Service port, it should be already defined. |


_Appears in:_
- [ApisixUpstreamSpec](#apisixupstreamspec)



#### UpstreamTimeout


UpstreamTimeout is settings for the read, send and connect to the upstream.



| Field | Description |
| --- | --- |
| `connect` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ |  |
| `send` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ |  |
| `read` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ |  |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

