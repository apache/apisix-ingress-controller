---
title: Custom Resource Definitions API Reference
slug: /reference/apisix-ingress-controller/crd-reference
description: Explore detailed reference documentation for the custom resource definitions (CRDs) supported by the APISIX Ingress Controller.
---

This document provides the API resource description for the API7 Ingress Controller custom resource definitions (CRDs).

## Packages
- [apisix.apache.org/v1alpha1](#apisixapacheorgv1alpha1)
- [apisix.apache.org/v2](#apisixapacheorgv2)


## apisix.apache.org/v1alpha1

Package v1alpha1 contains API Schema definitions for the apisix.apache.org v1alpha1 API group.

- [BackendTrafficPolicy](#backendtrafficpolicy)
- [Consumer](#consumer)
- [GatewayProxy](#gatewayproxy)
- [HTTPRoutePolicy](#httproutepolicy)
- [PluginConfig](#pluginconfig)
### BackendTrafficPolicy


BackendTrafficPolicy defines configuration for traffic handling policies applied to backend services.

<!-- BackendTrafficPolicy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `BackendTrafficPolicy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[BackendTrafficPolicySpec](#backendtrafficpolicyspec)_ | BackendTrafficPolicySpec defines traffic handling policies applied to backend services, such as load balancing strategy, connection settings, and failover behavior. |



### Consumer


Consumer defines configuration for a consumer.

<!-- Consumer resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `Consumer`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ConsumerSpec](#consumerspec)_ | ConsumerSpec defines configuration for a consumer, including consumer name, authentication credentials, and plugin settings. |



### GatewayProxy


GatewayProxy defines configuration for the gateway proxy instances used to route traffic to services.

<!-- GatewayProxy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `GatewayProxy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[GatewayProxySpec](#gatewayproxyspec)_ | GatewayProxySpec defines configuration of gateway proxy instances, including networking settings, global plugins, and plugin metadata. |



### HTTPRoutePolicy


HTTPRoutePolicy defines configuration of traffic policies.

<!-- HTTPRoutePolicy resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `HTTPRoutePolicy`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[HTTPRoutePolicySpec](#httproutepolicyspec)_ | HTTPRoutePolicySpec defines configuration of a HTTPRoutePolicy, including route priority and request matching conditions. |



### PluginConfig


PluginConfig defines plugin configuration.

<!-- PluginConfig resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v1alpha1`
| `kind` _string_ | `PluginConfig`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[PluginConfigSpec](#pluginconfigspec)_ | PluginConfigSpec defines the desired state of a PluginConfig, in which plugins and their configuration are specified. |



### Types

This section describes the types used by the CRDs.
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
| `scheme` _string_ | Scheme is the protocol used to communicate with the upstream. Default is `http`. Can be `http`, `https`, `grpc`, or `grpcs`. |
| `retries` _integer_ | Retries specify the number of times the gateway should retry sending requests when errors such as timeouts or 502 errors occur. |
| `timeout` _[Timeout](#timeout)_ | Timeout sets the read, send, and connect timeouts to the upstream. |
| `passHost` _string_ | PassHost configures how the host header should be determined when a request is forwarded to the upstream. Default is `pass`. Can be `pass`, `node` or `rewrite`:<br /> • `pass`: preserve the original Host header<br /> • `node`: use the upstream node’s host<br /> • `rewrite`: set to a custom host via `upstreamHost` |
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


ControlPlaneProvider defines configuration for control plane provider.



| Field | Description |
| --- | --- |
| `endpoints` _string array_ | Endpoints specifies the list of control plane endpoints. |
| `service` _[ProviderService](#providerservice)_ |  |
| `tlsVerify` _boolean_ | TlsVerify specifies whether to verify the TLS certificate of the control plane. |
| `auth` _[ControlPlaneAuth](#controlplaneauth)_ | Auth specifies the authentication configuration. |


_Appears in:_
- [GatewayProxyProvider](#gatewayproxyprovider)

#### Credential






| Field | Description |
| --- | --- |
| `type` _string_ | Type specifies the type of authentication to configure credentials for. Can be `jwt-auth`, `basic-auth`, `key-auth`, or `hmac-auth`. |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)_ | Config specifies the credential details for authentication. |
| `secretRef` _[SecretReference](#secretreference)_ | SecretRef references to the Secret that contains the credentials. |
| `name` _string_ | Name is the name of the credential. |


_Appears in:_
- [ConsumerSpec](#consumerspec)

#### GatewayProxyPlugin


GatewayProxyPlugin contains plugin configuration.



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
| `pluginMetadata` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io))_ | PluginMetadata configures common configuration shared by all plugin instances of the same name. |


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
| `priority` _integer_ | Priority sets the priority for route. when multiple routes have the same URI path, a higher value sets a higher priority in route matching. |
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
| `type` _string_ | Type specifies the load balancing algorithms to route traffic to the backend. Default is `roundrobin`. Can be `roundrobin`, `chash`, `ewma`, or `least_conn`. |
| `hashOn` _string_ | HashOn specified the type of field used for hashing, required when type is `chash`. Default is `vars`. Can be `vars`, `header`, `cookie`, `consumer`, or `vars_combinations`. |
| `key` _string_ | Key is used with HashOn, generally required when type is `chash`. When HashOn is `header` or `cookie`, specifies the name of the header or cookie. When HashOn is `consumer`, key is not required, as the consumer name is used automatically. When HashOn is `vars` or `vars_combinations`, key refers to one or a combination of [built-in variables](/enterprise/reference/built-in-variables). |


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
| `plugins` _[Plugin](#plugin) array_ | Plugins are an array of plugins and their configuration to be applied. |


_Appears in:_
- [PluginConfig](#pluginconfig)



#### ProviderService






| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the provider. |
| `port` _integer_ | Port is the port of the provider. |


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


ApisixConsumer defines configuration of a consumer and their authentication details.

<!-- ApisixConsumer resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixConsumer`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixConsumerSpec](#apisixconsumerspec)_ | ApisixConsumerSpec defines the consumer authentication configuration. |



### ApisixGlobalRule


ApisixGlobalRule defines configuration for global plugins.

<!-- ApisixGlobalRule resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixGlobalRule`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixGlobalRuleSpec](#apisixglobalrulespec)_ | ApisixGlobalRuleSpec defines the global plugin configuration. |



### ApisixPluginConfig


ApisixPluginConfig defines a reusable set of plugin configuration that can be referenced by routes.

<!-- ApisixPluginConfig resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixPluginConfig`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixPluginConfigSpec](#apisixpluginconfigspec)_ | ApisixPluginConfigSpec defines the plugin config configuration. |



### ApisixRoute


ApisixRoute is defines configuration for HTTP and stream routes.

<!-- ApisixRoute resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixRoute`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixRouteSpec](#apisixroutespec)_ | ApisixRouteSpec defines HTTP and stream route configuration. |



### ApisixTls


ApisixTls defines configuration for TLS and mutual TLS (mTLS).

<!-- ApisixTls resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixTls`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixTlsSpec](#apisixtlsspec)_ | ApisixTlsSpec defines the TLS configuration. |



### ApisixUpstream


ApisixUpstream defines configuration for upstream services.

<!-- ApisixUpstream resource -->

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `apisix.apache.org/v2`
| `kind` _string_ | `ApisixUpstream`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#objectmeta-v1-meta)_ | Please refer to the Kubernetes API documentation for details on the `metadata` field. |
| `spec` _[ApisixUpstreamSpec](#apisixupstreamspec)_ | ApisixUpstreamSpec defines the upstream configuration. |



### Types

This section describes the types used by the CRDs.
#### ActiveHealthCheck


ActiveHealthCheck defines the active upstream health check configuration.



| Field | Description |
| --- | --- |
| `type` _string_ | Type is the health check type. Can be `http`, `https`, or `tcp`. |
| `timeout` _[Duration](#duration)_ | Timeout sets health check timeout in seconds. |
| `concurrency` _integer_ | Concurrency sets the number of targets to be checked at the same time. |
| `host` _string_ | Host sets the upstream host. |
| `port` _integer_ | Port sets the upstream port. |
| `httpPath` _string_ | HTTPPath sets the HTTP probe request path. |
| `strictTLS` _boolean_ | StrictTLS sets whether to enforce TLS. |
| `requestHeaders` _string array_ | RequestHeaders sets the request headers. |
| `healthy` _[ActiveHealthCheckHealthy](#activehealthcheckhealthy)_ | Healthy configures the rules that define an upstream node as healthy. |
| `unhealthy` _[ActiveHealthCheckUnhealthy](#activehealthcheckunhealthy)_ | Unhealthy configures the rules that define an upstream node as unhealthy. |


_Appears in:_
- [HealthCheck](#healthcheck)

#### ActiveHealthCheckHealthy


UpstreamActiveHealthCheckHealthy defines the conditions used to actively determine whether an upstream node is healthy.



| Field | Description |
| --- | --- |
| `httpCodes` _integer array_ | HTTPCodes define a list of HTTP status codes that are considered healthy. |
| `successes` _integer_ | Successes define the number of successful probes to define a healthy target. |
| `interval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Interval defines the time interval for checking targets, in seconds. |


_Appears in:_
- [ActiveHealthCheck](#activehealthcheck)

#### ActiveHealthCheckUnhealthy


UpstreamActiveHealthCheckHealthy defines the conditions used to actively determine whether an upstream node is unhealthy.



| Field | Description |
| --- | --- |
| `httpCodes` _integer array_ | HTTPCodes define a list of HTTP status codes that are considered unhealthy. |
| `httpFailures` _integer_ | HTTPFailures define the number of HTTP failures to define an unhealthy target. |
| `tcpFailures` _integer_ | TCPFailures define the number of TCP failures to define an unhealthy target. |
| `timeout` _integer_ | Timeout sets health check timeout in seconds. |
| `interval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Interval defines the time interval for checking targets, in seconds. |


_Appears in:_
- [ActiveHealthCheck](#activehealthcheck)

#### ApisixConsumerAuthParameter






| Field | Description |
| --- | --- |
| `basicAuth` _[ApisixConsumerBasicAuth](#apisixconsumerbasicauth)_ | BasicAuth configures the basic authentication details. |
| `keyAuth` _[ApisixConsumerKeyAuth](#apisixconsumerkeyauth)_ | KeyAuth configures the key authentication details. |
| `wolfRBAC` _[ApisixConsumerWolfRBAC](#apisixconsumerwolfrbac)_ | WolfRBAC configures the Wolf RBAC authentication details. |
| `jwtAuth` _[ApisixConsumerJwtAuth](#apisixconsumerjwtauth)_ | JwtAuth configures the JWT authentication details. |
| `hmacAuth` _[ApisixConsumerHMACAuth](#apisixconsumerhmacauth)_ | HMACAuth configures the HMAC authentication details. |
| `ldapAuth` _[ApisixConsumerLDAPAuth](#apisixconsumerldapauth)_ | LDAPAuth configures the LDAP authentication details. |


_Appears in:_
- [ApisixConsumerSpec](#apisixconsumerspec)

#### ApisixConsumerBasicAuth


ApisixConsumerBasicAuth defines configuration for basic authentication.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ | SecretRef references a Kubernetes Secret containing the basic authentication credentials. |
| `value` _[ApisixConsumerBasicAuthValue](#apisixconsumerbasicauthvalue)_ | Value specifies the basic authentication credentials. |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerBasicAuthValue


ApisixConsumerBasicAuthValue defines the username and password configuration for basic authentication.



| Field | Description |
| --- | --- |
| `username` _string_ | Username is the basic authentication username. |
| `password` _string_ | Password is the basic authentication password. |


_Appears in:_
- [ApisixConsumerBasicAuth](#apisixconsumerbasicauth)

#### ApisixConsumerHMACAuth


ApisixConsumerHMACAuth defines configuration for the HMAC authentication.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ | SecretRef references a Kubernetes Secret containing the HMAC credentials. |
| `value` _[ApisixConsumerHMACAuthValue](#apisixconsumerhmacauthvalue)_ | Value specifies HMAC authentication credentials. |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerHMACAuthValue


ApisixConsumerHMACAuthValue defines configuration for HMAC authentication.



| Field | Description |
| --- | --- |
| `access_key` _string_ | AccessKey is the identifier used to look up the HMAC secret. |
| `secret_key` _string_ | SecretKey is the HMAC secret used to sign the request. |
| `algorithm` _string_ | Algorithm specifies the hashing algorithm (e.g., "hmac-sha256"). |
| `clock_skew` _integer_ | ClockSkew is the allowed time difference (in seconds) between client and server clocks. |
| `signed_headers` _string array_ | SignedHeaders lists the headers that must be included in the signature. |
| `keep_headers` _boolean_ | KeepHeaders determines whether the HMAC signature headers are preserved after verification. |
| `encode_uri_params` _boolean_ | EncodeURIParams indicates whether URI parameters are encoded when calculating the signature. |
| `validate_request_body` _boolean_ | ValidateRequestBody enables HMAC validation of the request body. |
| `max_req_body` _integer_ | MaxReqBody sets the maximum size (in bytes) of the request body that can be validated. |


_Appears in:_
- [ApisixConsumerHMACAuth](#apisixconsumerhmacauth)

#### ApisixConsumerJwtAuth


ApisixConsumerJwtAuth defines configuration for JWT authentication.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ | SecretRef references a Kubernetes Secret containing JWT authentication credentials. |
| `value` _[ApisixConsumerJwtAuthValue](#apisixconsumerjwtauthvalue)_ | Value specifies JWT authentication credentials. |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerJwtAuthValue


ApisixConsumerJwtAuthValue defines configuration for JWT authentication.



| Field | Description |
| --- | --- |
| `key` _string_ | Key is the unique identifier for the JWT credential. |
| `secret` _string_ | Secret is the shared secret used to sign the JWT (for symmetric algorithms). |
| `public_key` _string_ | PublicKey is the public key used to verify JWT signatures (for asymmetric algorithms). |
| `private_key` _string_ | PrivateKey is the private key used to sign the JWT (for asymmetric algorithms). |
| `algorithm` _string_ | Algorithm specifies the signing algorithm. Can be `HS256`, `HS384`, `HS512`, `RS256`, `RS384`, `RS512`, `ES256`, `ES384`, `ES512`, `PS256`, `PS384`, `PS512`, or `EdDSA`. Currently APISIX only supports `HS256`, `HS512`, `RS256`, and `ES256`. API7 Enterprise supports all algorithms. |
| `exp` _integer_ | Exp is the token expiration period in seconds. |
| `base64_secret` _boolean_ | Base64Secret indicates whether the secret is base64-encoded. |
| `lifetime_grace_period` _integer_ | LifetimeGracePeriod is the allowed clock skew in seconds for token expiration. |


_Appears in:_
- [ApisixConsumerJwtAuth](#apisixconsumerjwtauth)

#### ApisixConsumerKeyAuth


ApisixConsumerKeyAuth defines configuration for the key auth.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ | SecretRef references a Kubernetes Secret containing the key authentication credentials. |
| `value` _[ApisixConsumerKeyAuthValue](#apisixconsumerkeyauthvalue)_ | Value specifies the key authentication credentials. |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerKeyAuthValue


ApisixConsumerKeyAuthValue defines configuration for key authentication.



| Field | Description |
| --- | --- |
| `key` _string_ | Key is the credential used for key authentication. |


_Appears in:_
- [ApisixConsumerKeyAuth](#apisixconsumerkeyauth)

#### ApisixConsumerLDAPAuth


ApisixConsumerLDAPAuth defines configuration for the LDAP authentication.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ | SecretRef references a Kubernetes Secret containing the LDAP credentials. |
| `value` _[ApisixConsumerLDAPAuthValue](#apisixconsumerldapauthvalue)_ | Value specifies LDAP authentication credentials. |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerLDAPAuthValue


ApisixConsumerLDAPAuthValue defines configuration for LDAP authentication.



| Field | Description |
| --- | --- |
| `user_dn` _string_ | UserDN is the distinguished name (DN) of the LDAP user. |


_Appears in:_
- [ApisixConsumerLDAPAuth](#apisixconsumerldapauth)

#### ApisixConsumerSpec


ApisixConsumerSpec defines the desired state of ApisixConsumer.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. The controller uses this field to decide whether the resource should be managed. |
| `authParameter` _[ApisixConsumerAuthParameter](#apisixconsumerauthparameter)_ | AuthParameter defines the authentication credentials and configuration for this consumer. |


_Appears in:_
- [ApisixConsumer](#apisixconsumer)

#### ApisixConsumerWolfRBAC


ApisixConsumerWolfRBAC defines configuration for the Wolf RBAC authentication.



| Field | Description |
| --- | --- |
| `secretRef` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#localobjectreference-v1-core)_ | SecretRef references a Kubernetes Secret containing the Wolf RBAC token. |
| `value` _[ApisixConsumerWolfRBACValue](#apisixconsumerwolfrbacvalue)_ | Value specifies the Wolf RBAC token. |


_Appears in:_
- [ApisixConsumerAuthParameter](#apisixconsumerauthparameter)

#### ApisixConsumerWolfRBACValue


ApisixConsumerWolfRBACValue defines configuration for Wolf RBAC authentication.



| Field | Description |
| --- | --- |
| `server` _string_ | Server is the URL of the Wolf RBAC server. |
| `appid` _string_ | Appid is the application identifier used when communicating with the Wolf RBAC server. |
| `header_prefix` _string_ | HeaderPrefix is the prefix added to request headers for RBAC enforcement. |


_Appears in:_
- [ApisixConsumerWolfRBAC](#apisixconsumerwolfrbac)

#### ApisixGlobalRuleSpec


ApisixGlobalRuleSpec defines configuration for global plugins.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. The controller uses this field to decide whether the resource should be managed. |
| `plugins` _[ApisixRoutePlugin](#apisixrouteplugin) array_ | Plugins contain a list of global plugins. |


_Appears in:_
- [ApisixGlobalRule](#apisixglobalrule)

#### ApisixMutualTlsClientConfig


ApisixMutualTlsClientConfig describes the mutual TLS CA and verification settings.



| Field | Description |
| --- | --- |
| `caSecret` _[ApisixSecret](#apisixsecret)_ | CASecret references the secret containing the CA certificate for client certificate validation. |
| `depth` _integer_ | Depth specifies the maximum verification depth for the client certificate chain. |
| `skip_mtls_uri_regex` _string array_ | SkipMTLSUriRegex contains RegEx patterns for URIs to skip mutual TLS verification. |


_Appears in:_
- [ApisixTlsSpec](#apisixtlsspec)

#### ApisixPluginConfigSpec


ApisixPluginConfigSpec defines the desired state of ApisixPluginConfigSpec.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. The controller uses this field to decide whether the resource should be managed. |
| `plugins` _[ApisixRoutePlugin](#apisixrouteplugin) array_ | Plugins contain a list of plugins. |


_Appears in:_
- [ApisixPluginConfig](#apisixpluginconfig)

#### ApisixRouteAuthentication


ApisixRouteAuthentication represents authentication-related configuration in ApisixRoute.



| Field | Description |
| --- | --- |
| `enable` _boolean_ | Enable toggles authentication on or off. |
| `type` _string_ | Type specifies the authentication type. |
| `keyAuth` _[ApisixRouteAuthenticationKeyAuth](#apisixrouteauthenticationkeyauth)_ | KeyAuth defines configuration for key authentication. |
| `jwtAuth` _[ApisixRouteAuthenticationJwtAuth](#apisixrouteauthenticationjwtauth)_ | JwtAuth defines configuration for JWT authentication. |
| `ldapAuth` _[ApisixRouteAuthenticationLDAPAuth](#apisixrouteauthenticationldapauth)_ | LDAPAuth defines configuration for LDAP authentication. |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)

#### ApisixRouteAuthenticationJwtAuth


ApisixRouteAuthenticationJwtAuth defines JWT authentication configuration in ApisixRouteAuthentication.



| Field | Description |
| --- | --- |
| `header` _string_ | Header specifies the HTTP header name to look for the JWT token. |
| `query` _string_ | Query specifies the URL query parameter name to look for the JWT token. |
| `cookie` _string_ | Cookie specifies the cookie name to look for the JWT token. |


_Appears in:_
- [ApisixRouteAuthentication](#apisixrouteauthentication)

#### ApisixRouteAuthenticationKeyAuth


ApisixRouteAuthenticationKeyAuth defines key authentication configuration in ApisixRouteAuthentication.



| Field | Description |
| --- | --- |
| `header` _string_ | Header specifies the HTTP header name to look for the key authentication token. |


_Appears in:_
- [ApisixRouteAuthentication](#apisixrouteauthentication)

#### ApisixRouteAuthenticationLDAPAuth


ApisixRouteAuthenticationLDAPAuth defines LDAP authentication configuration in ApisixRouteAuthentication.



| Field | Description |
| --- | --- |
| `base_dn` _string_ | BaseDN is the base distinguished name (DN) for LDAP searches. |
| `ldap_uri` _string_ | LDAPURI is the URI of the LDAP server. |
| `use_tls` _boolean_ | UseTLS indicates whether to use TLS for the LDAP connection. |
| `uid` _string_ | UID is the user identifier attribute in LDAP. |


_Appears in:_
- [ApisixRouteAuthentication](#apisixrouteauthentication)

#### ApisixRouteHTTP


ApisixRouteHTTP represents a single HTTP route configuration.



| Field | Description |
| --- | --- |
| `name` _string_ | Name is the unique rule name and cannot be empty. |
| `priority` _integer_ | Priority defines the route priority when multiple routes share the same URI path. Higher values mean higher priority in route matching. |
| `timeout` _[UpstreamTimeout](#upstreamtimeout)_ | Timeout specifies upstream timeout settings. |
| `match` _[ApisixRouteHTTPMatch](#apisixroutehttpmatch)_ | Match defines the HTTP request matching criteria. |
| `backends` _[ApisixRouteHTTPBackend](#apisixroutehttpbackend) array_ | Backends lists potential backend services to proxy requests to. If more than one backend is specified, the `traffic-split` plugin is used to distribute traffic according to backend weights. |
| `upstreams` _[ApisixRouteUpstreamReference](#apisixrouteupstreamreference) array_ | Upstreams references ApisixUpstream CRDs. |
| `websocket` _boolean_ | Websocket enables or disables websocket support for this route. |
| `plugin_config_name` _string_ | PluginConfigName specifies the name of the plugin config to apply. |
| `plugin_config_namespace` _string_ | PluginConfigNamespace specifies the namespace of the plugin config. Defaults to the namespace of the ApisixRoute if not set. |
| `plugins` _[ApisixRoutePlugin](#apisixrouteplugin) array_ | Plugins lists additional plugins applied to this route. |
| `authentication` _[ApisixRouteAuthentication](#apisixrouteauthentication)_ | Authentication holds authentication-related configuration for this route. |


_Appears in:_
- [ApisixRouteSpec](#apisixroutespec)

#### ApisixRouteHTTPBackend


ApisixRouteHTTPBackend represents an HTTP backend (Kubernetes Service).



| Field | Description |
| --- | --- |
| `serviceName` _string_ | ServiceName is the name of the Kubernetes Service. Cross-namespace references are not supported—ensure the ApisixRoute and the Service are in the same namespace. |
| `servicePort` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#intorstring-intstr-util)_ | ServicePort is the port of the Kubernetes Service. This can be either the port name or port number. |
| `resolveGranularity` _string_ | ResolveGranularity determines how the backend service is resolved. Valid values are `endpoints` and `service`. When set to `endpoints`, individual pod IPs will be used; otherwise, the Service's ClusterIP or ExternalIP is used. The default is `endpoints`. |
| `weight` _integer_ | Weight specifies the relative traffic weight for this backend. |
| `subset` _string_ | Subset specifies a named subset of the target Service. The subset must be pre-defined in the corresponding ApisixUpstream resource. |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)

#### ApisixRouteHTTPMatch


ApisixRouteHTTPMatch defines the conditions used to match incoming HTTP requests.



| Field | Description |
| --- | --- |
| `paths` _string array_ | Paths is a list of URI path patterns to match. At least one path must be specified. Supports exact matches and prefix matches. For prefix matches, append `*` to the path, such as `/foo*`. |
| `methods` _string array_ | Methods specifies the HTTP methods to match. |
| `hosts` _string array_ | Hosts specifies Host header values to match. Supports exact and wildcard domains. Only one level of wildcard is allowed (e.g., `*.example.com` is valid, but `*.*.example.com` is not). |
| `remoteAddrs` _string array_ | RemoteAddrs is a list of source IP addresses or CIDR ranges to match. Supports both IPv4 and IPv6 formats. |
| `exprs` _[ApisixRouteHTTPMatchExprs](#apisixroutehttpmatchexprs)_ | NginxVars defines match conditions based on Nginx variables. |
| `filter_func` _string_ | FilterFunc is a user-defined function for advanced request filtering. The function can use Nginx variables through the `vars` parameter. This field is supported in APISIX but not in API7 Enterprise. |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)

#### ApisixRouteHTTPMatchExpr


ApisixRouteHTTPMatchExpr represents a binary expression used to match requests based on Nginx variables.



| Field | Description |
| --- | --- |
| `subject` _[ApisixRouteHTTPMatchExprSubject](#apisixroutehttpmatchexprsubject)_ | Subject defines the left-hand side of the expression. It can be any [built-in variable](/apisix/reference/built-in-variables) or string literal. |
| `op` _string_ | Op specifies the operator used in the expression. Can be `Equal`, `NotEqual`, `GreaterThan`, `GreaterThanEqual`, `LessThan`, `LessThanEqual`, `RegexMatch`, `RegexNotMatch`, `RegexMatchCaseInsensitive`, `RegexNotMatchCaseInsensitive`, `In`, or `NotIn`. |
| `set` _string array_ | Set provides a list of acceptable values for the expression. This should be used when Op is `In` or `NotIn`. |
| `value` _string_ | Value defines a single value to compare against the subject. This should be used when Op is not `In` or `NotIn`. Set and Value are mutually exclusive—only one should be set at a time. |


_Appears in:_
- [ApisixRouteHTTPMatchExprs](#apisixroutehttpmatchexprs)

#### ApisixRouteHTTPMatchExprSubject


ApisixRouteHTTPMatchExprSubject describes the subject of a route matching expression.



| Field | Description |
| --- | --- |
| `scope` _string_ | Scope specifies the subject scope and can be `Header`, `Query`, or `Path`. When Scope is `Path`, Name will be ignored. |
| `name` _string_ | Name is the name of the header or query parameter. |


_Appears in:_
- [ApisixRouteHTTPMatchExpr](#apisixroutehttpmatchexpr)

#### ApisixRouteHTTPMatchExprs
_Base type:_ `[ApisixRouteHTTPMatchExpr](#apisixroutehttpmatchexpr)`





| Field | Description |
| --- | --- |
| `subject` _[ApisixRouteHTTPMatchExprSubject](#apisixroutehttpmatchexprsubject)_ | Subject defines the left-hand side of the expression. It can be any [built-in variable](/apisix/reference/built-in-variables) or string literal. |
| `op` _string_ | Op specifies the operator used in the expression. Can be `Equal`, `NotEqual`, `GreaterThan`, `GreaterThanEqual`, `LessThan`, `LessThanEqual`, `RegexMatch`, `RegexNotMatch`, `RegexMatchCaseInsensitive`, `RegexNotMatchCaseInsensitive`, `In`, or `NotIn`. |
| `set` _string array_ | Set provides a list of acceptable values for the expression. This should be used when Op is `In` or `NotIn`. |
| `value` _string_ | Value defines a single value to compare against the subject. This should be used when Op is not `In` or `NotIn`. Set and Value are mutually exclusive—only one should be set at a time. |


_Appears in:_
- [ApisixRouteHTTPMatch](#apisixroutehttpmatch)

#### ApisixRoutePlugin


ApisixRoutePlugin represents an APISIX plugin.



| Field | Description |
| --- | --- |
| `name` _string_ | The plugin name. |
| `enable` _boolean_ | Whether this plugin is in use, default is true. |
| `config` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#json-v1-apiextensions-k8s-io)_ | Plugin configuration. |
| `secretRef` _string_ | Plugin configuration secretRef. |


_Appears in:_
- [ApisixGlobalRuleSpec](#apisixglobalrulespec)
- [ApisixPluginConfigSpec](#apisixpluginconfigspec)
- [ApisixRouteHTTP](#apisixroutehttp)
- [ApisixRouteStream](#apisixroutestream)



#### ApisixRouteSpec


ApisixRouteSpec is the spec definition for ApisixRoute.
It defines routing rules for both HTTP and stream traffic.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of the IngressClass this route belongs to. It allows multiple controllers to watch and reconcile different routes. |
| `http` _[ApisixRouteHTTP](#apisixroutehttp) array_ | HTTP defines a list of HTTP route rules. Each rule specifies conditions to match HTTP requests and how to forward them. |
| `stream` _[ApisixRouteStream](#apisixroutestream) array_ | Stream defines a list of stream route rules. Each rule specifies conditions to match TCP/UDP traffic and how to forward them. |


_Appears in:_
- [ApisixRoute](#apisixroute)

#### ApisixRouteStream


ApisixRouteStream defines the configuration for a Layer 4 (TCP/UDP) route.



| Field | Description |
| --- | --- |
| `name` _string_ | Name is a unique identifier for the route. This field must not be empty. |
| `protocol` _string_ | Protocol specifies the L4 protocol to match. Can be `tcp` or `udp`. |
| `match` _[ApisixRouteStreamMatch](#apisixroutestreammatch)_ | Match defines the criteria used to match incoming TCP or UDP connections. |
| `backend` _[ApisixRouteStreamBackend](#apisixroutestreambackend)_ | Backend specifies the destination service to which traffic should be forwarded. |
| `plugins` _[ApisixRoutePlugin](#apisixrouteplugin) array_ | Plugins defines a list of plugins to apply to this route. |


_Appears in:_
- [ApisixRouteSpec](#apisixroutespec)

#### ApisixRouteStreamBackend


ApisixRouteStreamBackend represents the backend service for a TCP or UDP stream route.



| Field | Description |
| --- | --- |
| `serviceName` _string_ | ServiceName is the name of the Kubernetes Service. Cross-namespace references are not supported—ensure the ApisixRoute and the Service are in the same namespace. |
| `servicePort` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#intorstring-intstr-util)_ | ServicePort is the port of the Kubernetes Service. This can be either the port name or port number. |
| `resolveGranularity` _string_ | ResolveGranularity determines how the backend service is resolved. Valid values are `endpoints` and `service`. When set to `endpoints`, individual pod IPs will be used; otherwise, the Service's ClusterIP or ExternalIP is used. The default is `endpoints`. |
| `subset` _string_ | Subset specifies a named subset of the target Service. The subset must be pre-defined in the corresponding ApisixUpstream resource. |


_Appears in:_
- [ApisixRouteStream](#apisixroutestream)

#### ApisixRouteStreamMatch


ApisixRouteStreamMatch represents the matching conditions for a stream route.



| Field | Description |
| --- | --- |
| `ingressPort` _integer_ | IngressPort is the port on which the APISIX Ingress proxy server listens. This must be a statically configured port, as APISIX does not support dynamic port binding. |
| `host` _string_ | Host is the destination host address used to match the incoming TCP/UDP traffic. |


_Appears in:_
- [ApisixRouteStream](#apisixroutestream)

#### ApisixRouteUpstreamReference


ApisixRouteUpstreamReference references an ApisixUpstream CRD to be used as a backend.
It can be used in traffic-splitting scenarios or to select a specific upstream configuration.



| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the ApisixUpstream resource. |
| `weight` _integer_ | Weight is the weight assigned to this upstream. |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)

#### ApisixSecret


ApisixSecret describes a reference to a Kubernetes Secret, including its name and namespace.
This is used to locate secrets such as certificates or credentials for plugins or TLS configuration.



| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the Kubernetes Secret. |
| `namespace` _string_ | Namespace is the namespace where the Kubernetes Secret is located. |


_Appears in:_
- [ApisixMutualTlsClientConfig](#apisixmutualtlsclientconfig)
- [ApisixTlsSpec](#apisixtlsspec)
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)















#### ApisixTlsSpec


ApisixTlsSpec defines configurations for TLS and mutual TLS.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName specifies which IngressClass this resource is associated with. The APISIX controller only processes this resource if the class matches its own. |
| `hosts` _[HostType](#hosttype) array_ | Hosts lists the SNI (Server Name Indication) hostnames that this TLS configuration applies to. Must contain at least one host. |
| `secret` _[ApisixSecret](#apisixsecret)_ | Secret refers to the Kubernetes TLS secret containing the certificate and private key. This secret must exist in the specified namespace and contain valid TLS data. |
| `client` _[ApisixMutualTlsClientConfig](#apisixmutualtlsclientconfig)_ | Client defines mutual TLS (mTLS) settings, such as the CA certificate and verification depth. |


_Appears in:_
- [ApisixTls](#apisixtls)

#### ApisixUpstreamConfig


ApisixUpstreamConfig defines configuration for upstream services.



| Field | Description |
| --- | --- |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer specifies the load balancer configuration for Kubernetes Service. |
| `scheme` _string_ | Scheme is the protocol used to communicate with the upstream. Default is `http`. Can be `http`, `https`, `grpc`, or `grpcs`. |
| `retries` _integer_ | Retries defines the number of retry attempts APISIX should make when a failure occurs. Failures include timeouts, network errors, or 5xx status codes. |
| `timeout` _[UpstreamTimeout](#upstreamtimeout)_ | Timeout specifies the connection, send, and read timeouts for upstream requests. |
| `healthCheck` _[HealthCheck](#healthcheck)_ | HealthCheck defines the active and passive health check configuration for the upstream. Deprecated: no longer supported in standalone mode. |
| `tlsSecret` _[ApisixSecret](#apisixsecret)_ | TLSSecret references a Kubernetes Secret that contains the client certificate and key for mutual TLS when connecting to the upstream. |
| `subsets` _[ApisixUpstreamSubset](#apisixupstreamsubset) array_ | Subsets defines labeled subsets of service endpoints, typically used for service versioning or canary deployments. |
| `passHost` _string_ | PassHost configures how the host header should be determined when a request is forwarded to the upstream. Default is `pass`. Can be `pass`, `node` or `rewrite`:<br /> • `pass`: preserve the original Host header<br /> • `node`: use the upstream node’s host<br /> • `rewrite`: set to a custom host via upstreamHost |
| `upstreamHost` _string_ | UpstreamHost sets a custom Host header when passHost is set to `rewrite`. |
| `discovery` _[Discovery](#discovery)_ | Discovery configures service discovery for the upstream. Deprecated: no longer supported in standalone mode. |


_Appears in:_
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### ApisixUpstreamExternalNode


ApisixUpstreamExternalNode defines configuration for an external upstream node.
This allows referencing services outside the cluster.



| Field | Description |
| --- | --- |
| `name` _string_ | Name is the hostname or IP address of the external node. |
| `type` _[ApisixUpstreamExternalType](#apisixupstreamexternaltype)_ | Type indicates the kind of external node. Can be `Domain`, or `Service`. |
| `weight` _integer_ | Weight defines the load balancing weight of this node. Higher values increase the share of traffic sent to this node. |
| `port` _integer_ | Port specifies the port number on which the external node is accepting traffic. |


_Appears in:_
- [ApisixUpstreamSpec](#apisixupstreamspec)

#### ApisixUpstreamExternalType
_Base type:_ `string`

ApisixUpstreamExternalType is the external service type





_Appears in:_
- [ApisixUpstreamExternalNode](#apisixupstreamexternalnode)

#### ApisixUpstreamSpec


ApisixUpstreamSpec describes the desired configuration of an ApisixUpstream resource.
It defines how traffic should be routed to backend services, including upstream node
definitions and custom configuration.



| Field | Description |
| --- | --- |
| `ingressClassName` _string_ | IngressClassName is the name of an IngressClass cluster resource. Controller implementations use this field to determine whether they should process this ApisixUpstream resource. |
| `externalNodes` _[ApisixUpstreamExternalNode](#apisixupstreamexternalnode) array_ | ExternalNodes defines a static list of backend nodes located outside the cluster. When this field is set, the upstream will route traffic directly to these nodes without DNS resolution or service discovery. |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer specifies the load balancer configuration for Kubernetes Service. |
| `scheme` _string_ | Scheme is the protocol used to communicate with the upstream. Default is `http`. Can be `http`, `https`, `grpc`, or `grpcs`. |
| `retries` _integer_ | Retries defines the number of retry attempts APISIX should make when a failure occurs. Failures include timeouts, network errors, or 5xx status codes. |
| `timeout` _[UpstreamTimeout](#upstreamtimeout)_ | Timeout specifies the connection, send, and read timeouts for upstream requests. |
| `healthCheck` _[HealthCheck](#healthcheck)_ | HealthCheck defines the active and passive health check configuration for the upstream. Deprecated: no longer supported in standalone mode. |
| `tlsSecret` _[ApisixSecret](#apisixsecret)_ | TLSSecret references a Kubernetes Secret that contains the client certificate and key for mutual TLS when connecting to the upstream. |
| `subsets` _[ApisixUpstreamSubset](#apisixupstreamsubset) array_ | Subsets defines labeled subsets of service endpoints, typically used for service versioning or canary deployments. |
| `passHost` _string_ | PassHost configures how the host header should be determined when a request is forwarded to the upstream. Default is `pass`. Can be `pass`, `node` or `rewrite`:<br /> • `pass`: preserve the original Host header<br /> • `node`: use the upstream node’s host<br /> • `rewrite`: set to a custom host via upstreamHost |
| `upstreamHost` _string_ | UpstreamHost sets a custom Host header when passHost is set to `rewrite`. |
| `discovery` _[Discovery](#discovery)_ | Discovery configures service discovery for the upstream. Deprecated: no longer supported in standalone mode. |
| `portLevelSettings` _[PortLevelSettings](#portlevelsettings) array_ | PortLevelSettings allows fine-grained upstream configuration for specific ports, useful when a backend service exposes multiple ports with different behaviors or protocols. |


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


Discovery defines the service discovery configuration for dynamically resolving upstream nodes.
This is used when APISIX integrates with a service registry such as Nacos, Consul, or Eureka.



| Field | Description |
| --- | --- |
| `serviceName` _string_ | ServiceName is the name of the service to discover. |
| `type` _string_ | Type is the name of the service discovery provider. |
| `args` _object (keys:string, values:string)_ | Args contains additional configuration parameters required by the discovery provider. These are passed as key-value pairs. |


_Appears in:_
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### HealthCheck


HealthCheck defines the health check configuration for upstream nodes.
It includes active checks (proactively probing the nodes) and optional passive checks (monitoring based on traffic).



| Field | Description |
| --- | --- |
| `active` _[ActiveHealthCheck](#activehealthcheck)_ | Active health checks proactively send requests to upstream nodes to determine their availability. |
| `passive` _[PassiveHealthCheck](#passivehealthcheck)_ | Passive health checks evaluate upstream health based on observed traffic, such as timeouts or errors. |


_Appears in:_
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### HostType
_Base type:_ `string`







_Appears in:_
- [ApisixTlsSpec](#apisixtlsspec)

#### LoadBalancer


LoadBalancer defines the load balancing strategy for distributing traffic across upstream nodes.



| Field | Description |
| --- | --- |
| `type` _string_ | Type specifies the load balancing algorithms to route traffic to the backend. Default is `roundrobin`. Can be `roundrobin`, `chash`, `ewma`, or `least_conn`. |
| `hashOn` _string_ | HashOn specified the type of field used for hashing, required when type is `chash`. Default is `vars`. Can be `vars`, `header`, `cookie`, `consumer`, or `vars_combinations`. |
| `key` _string_ | Key is used with HashOn, generally required when type is `chash`. When HashOn is `header` or `cookie`, specifies the name of the header or cookie. When HashOn is `consumer`, key is not required, as the consumer name is used automatically. When HashOn is `vars` or `vars_combinations`, key refers to one or a combination of [built-in variables](/enterprise/reference/built-in-variables). |


_Appears in:_
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

#### PassiveHealthCheck


PassiveHealthCheck defines the conditions used to determine whether
an upstream node is healthy or unhealthy based on passive observations.
Passive health checks rely on real traffic responses instead of active probes.



| Field | Description |
| --- | --- |
| `type` _string_ | Type specifies the type of passive health check. Can be `http`, `https`, or `tcp`. |
| `healthy` _[PassiveHealthCheckHealthy](#passivehealthcheckhealthy)_ | Healthy defines the conditions under which an upstream node is considered healthy. |
| `unhealthy` _[PassiveHealthCheckUnhealthy](#passivehealthcheckunhealthy)_ | Unhealthy defines the conditions under which an upstream node is considered unhealthy. |


_Appears in:_
- [HealthCheck](#healthcheck)

#### PassiveHealthCheckHealthy


PassiveHealthCheckHealthy defines the conditions used to passively determine whether an upstream node is healthy.



| Field | Description |
| --- | --- |
| `httpCodes` _integer array_ | HTTPCodes define a list of HTTP status codes that are considered healthy. |
| `successes` _integer_ | Successes define the number of successful probes to define a healthy target. |


_Appears in:_
- [ActiveHealthCheckHealthy](#activehealthcheckhealthy)
- [PassiveHealthCheck](#passivehealthcheck)

#### PassiveHealthCheckUnhealthy


UpstreamPassiveHealthCheckUnhealthy defines the conditions used to passively determine whether an upstream node is unhealthy.



| Field | Description |
| --- | --- |
| `httpCodes` _integer array_ | HTTPCodes define a list of HTTP status codes that are considered unhealthy. |
| `httpFailures` _integer_ | HTTPFailures define the number of HTTP failures to define an unhealthy target. |
| `tcpFailures` _integer_ | TCPFailures define the number of TCP failures to define an unhealthy target. |
| `timeout` _integer_ | Timeout sets health check timeout in seconds. |


_Appears in:_
- [ActiveHealthCheckUnhealthy](#activehealthcheckunhealthy)
- [PassiveHealthCheck](#passivehealthcheck)

#### PortLevelSettings


PortLevelSettings configures the ApisixUpstreamConfig for each individual port. It inherits
configuration from the outer level (the whole Kubernetes Service) and overrides some of
them if they are set on the port level.



| Field | Description |
| --- | --- |
| `loadbalancer` _[LoadBalancer](#loadbalancer)_ | LoadBalancer specifies the load balancer configuration for Kubernetes Service. |
| `scheme` _string_ | Scheme is the protocol used to communicate with the upstream. Default is `http`. Can be `http`, `https`, `grpc`, or `grpcs`. |
| `retries` _integer_ | Retries defines the number of retry attempts APISIX should make when a failure occurs. Failures include timeouts, network errors, or 5xx status codes. |
| `timeout` _[UpstreamTimeout](#upstreamtimeout)_ | Timeout specifies the connection, send, and read timeouts for upstream requests. |
| `healthCheck` _[HealthCheck](#healthcheck)_ | HealthCheck defines the active and passive health check configuration for the upstream. Deprecated: no longer supported in standalone mode. |
| `tlsSecret` _[ApisixSecret](#apisixsecret)_ | TLSSecret references a Kubernetes Secret that contains the client certificate and key for mutual TLS when connecting to the upstream. |
| `subsets` _[ApisixUpstreamSubset](#apisixupstreamsubset) array_ | Subsets defines labeled subsets of service endpoints, typically used for service versioning or canary deployments. |
| `passHost` _string_ | PassHost configures how the host header should be determined when a request is forwarded to the upstream. Default is `pass`. Can be `pass`, `node` or `rewrite`:<br /> • `pass`: preserve the original Host header<br /> • `node`: use the upstream node’s host<br /> • `rewrite`: set to a custom host via upstreamHost |
| `upstreamHost` _string_ | UpstreamHost sets a custom Host header when passHost is set to `rewrite`. |
| `discovery` _[Discovery](#discovery)_ | Discovery configures service discovery for the upstream. Deprecated: no longer supported in standalone mode. |
| `port` _integer_ | Port is a Kubernetes Service port. |


_Appears in:_
- [ApisixUpstreamSpec](#apisixupstreamspec)





#### UpstreamTimeout


UpstreamTimeout defines timeout settings for connecting, sending, and reading from the upstream.



| Field | Description |
| --- | --- |
| `connect` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Connect timeout for establishing a connection to the upstream. |
| `send` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Send timeout for sending data to the upstream. |
| `read` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#duration-v1-meta)_ | Read timeout for reading data from the upstream. |


_Appears in:_
- [ApisixRouteHTTP](#apisixroutehttp)
- [ApisixUpstreamConfig](#apisixupstreamconfig)
- [ApisixUpstreamSpec](#apisixupstreamspec)
- [PortLevelSettings](#portlevelsettings)

