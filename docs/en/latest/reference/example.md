---
title: Configuration Examples
slug: /reference/apisix-ingress-controller/examples
description: Explore a variety of APISIX Ingress Controller configuration examples to help you customize settings to suit your environment effectively.
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

APISIX Ingress Controller supports both Ingress resources and Gateway API for traffic management in Kubernetes. In addition to these standard Kubernetes APIs, the APISIX Ingress Controller also supports a set of [CRDs (Custom Resource Definitions)](./api-reference.md) designed specifically for APISIX-native functionality.

This document provides examples of common configurations covering how and when to use these resources. You should adjust custom values such as namespaces, route URIs, and credentials to match your environment.

## Configure CP Endpoint and Admin Key

To update the Control Plane endpoint and admin key for connectivity between APISIX Ingress Controller and Control Plane at runtime:

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-config
  namespace: apisix-ingress
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - http://127.0.0.1:9180
      auth:
        type: AdminKey
        adminKey:
          value: replace-with-your-admin-key
```

## Define Controller and Gateway

To specify the controller responsible for handling resources before applying further configurations:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'Ingress', value: 'ingress'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: apisix
spec:
  controllerName: "apisix.apache.org/apisix-ingress-controller"
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
spec:
  gatewayClassName: apisix
  listeners:
  - name: http
    protocol: HTTP
    port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-config
```

</TabItem>

<TabItem value="ingress">

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: apisix.apache.org/apisix-ingress-controller
  parameters:
    apiGroup: apisix.apache.org
    kind: GatewayProxy
    name: apisix-config
    namespace: apisix-ingress
    scope: Namespace
```

</TabItem>

<TabItem value="apisix-crd">

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: apisix.apache.org/apisix-ingress-controller
  parameters:
    apiGroup: apisix.apache.org
    kind: GatewayProxy
    name: apisix-config
    namespace: apisix-ingress
    scope: Namespace
```

</TabItem>

</Tabs>

## Route to K8s Services

To create a route that proxies requests to a service on K8s:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'Ingress', value: 'ingress'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  rules:
  - matches:
    - path:
        type: Exact
        value: /ip
    backendRefs:
    - name: httpbin
      port: 80
```

</TabItem>

<TabItem value="ingress">

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin
spec:
  ingressClassName: apisix
  rules:
  - http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: httpbin
            port:
              number: 80
```

</TabItem>

<TabItem value="apisix-crd">

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin
spec:
  ingressClassName: apisix
  http:
  - name: httpbin
    match:
      paths:
      - /ip
    backends:
    - serviceName: httpbin
      servicePort: 80
```

</TabItem>

</Tabs>

## Route to External Services

To create a route that proxies requests to a service publicly hosted:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'Ingress', value: 'ingress'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: httpbin.org
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: get-ip
spec:
  parentRefs:
  - name: apisix
  rules:
  - matches:
    - path:
        type: Exact
        value: /ip
    backendRefs:
    - name: httpbin-external-domain
      port: 80
```

</TabItem>

<TabItem value="ingress">

```yaml
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: httpbin.org
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: get-ip
spec:
  rules:
  - http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: httpbin-external-domain
            port:
              number: 80
```

</TabItem>

<TabItem value="apisix-crd">

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-external-domain
spec:
  externalNodes:
  - type: Domain
    name: httpbin.org
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: get-ip
spec:
  ingressClassName: apisix
  http:
    - name: get-ip
      match:
        paths:
          - /ip
      upstreams:
      - name: httpbin-external-domain
```

</TabItem>

</Tabs>

## Configure Weighted Services

To create a route that proxies traffic to upstream services by weight:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  rules:
  - matches:
    - path:
        type: Exact
        value: /ip
    backendRefs:
    - name: httpbin-1
      port: 80
      weight: 3
    - name: httpbin-2
      port: 80
      weight: 7
```

</TabItem>

<TabItem value="apisix-crd">

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin
spec:
  ingressClassName: apisix
  http:
  - name: httpbin
    match:
      paths:
      - /ip
    backends:
    - serviceName: httpbin-1
      servicePort: 80
      weight: 3
    - serviceName: httpbin-2
      servicePort: 80
      weight: 7
```

</TabItem>

</Tabs>

This configuration is not supported by the Ingress resource.

## Configure Upstream

To configure upstream related configurations, including load balancing algorithm, how the host header is passed to upstream, service timeout, and more:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin
    kind: Service
    group: ""
  timeout:
    send: 10s
    read: 10s
    connect: 10s
  scheme: http
  retries: 10
  loadbalancer:
    type: roundrobin
  passHost: rewrite
  upstreamHost: httpbin.example.com
```

</TabItem>

<TabItem value="apisix-crd">

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  ingressClassName: apisix
  timeout:
    send: 10s
    read: 10s
    connect: 10s
  scheme: http
  retries: 10
  loadbalancer:
    type: roundrobin
  passHost: rewrite
  upstreamHost: httpbin.example.com
```

</TabItem>

</Tabs>

## Configure Consumer and Credentials

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

To create a consumer and configure the authentication credentials directly on the consumer:

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: alice
spec:
  gatewayRef:
    name: apisix
  credentials:
    - type: key-auth
      name: primary-key
      config:
        key: alice-primary-key
```

You can also use the secret CRD, where the credential should be base64 encoded:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: key-auth-primary
data:
  key: YWxpY2UtcHJpbWFyeS1rZXk=
---
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: alice
spec:
  gatewayRef:
    name: apisix
  credentials:
    - type: key-auth
      name: key-auth-primary
      secretRef:
        name: key-auth-primary
```

</TabItem>

<TabItem value="apisix-crd">

To create a consumer and configure the authentication credentials directly on the consumer:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: alice
spec:
  ingressClassName: apisix
  authParameter:
    keyAuth:
      value:
        key: alice-primary-key
```

You can also use the secret CRD, where the credential should be base64 encoded:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: key-auth-primary
data:
  key: YWxpY2UtcHJpbWFyeS1rZXk=
---
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: alice
spec:
  ingressClassName: apisix
  authParameter:
    keyAuth:
      secretRef:
        name: key-auth-primary
```

</TabItem>

</Tabs>

## Configure Plugin on Consumer

To configure plugin(s) on a consumer, such as a rate limiting plugin:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: alice
spec:
  gatewayRef:
    name: apisix
  credentials:
    - type: key-auth
      name: alice-key
      config:
        key: alice-key
  plugins:
    - name: limit-count
      config:
        count: 3
        time_window: 60
        key: remote_addr
        key_type: var
        policy: local
        rejected_code: 429
        rejected_msg: Too many requests
        show_limit_quota_header: true
        allow_degradation: false
```

</TabItem>

<TabItem value="apisix-crd">

ApisixConsumer currently does not support configuring plugins on consumers.

</TabItem>

</Tabs>

## Configure Route Priority and Matching Conditions

To configure route priority and request matching conditions on a targeted route:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  priority: 10
  vars:
  - - http_x_test_name
    - ==
    - new_name
  - - arg_test
    - ==
    - test_name
```

</TabItem>

<TabItem value="apisix-crd">

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin
spec:
  ingressClassName: apisix
  http:
  - name: httpbin
    match:
      paths:
      - /*
      exprs:
      - subject:
          scope: Header
          name: X-Test-Name
        op: Equal
        value: new_name
      - subject:
          scope: Query
          name: test
        op: Equal
        value: test_name
    backends:
    - serviceName: httpbin
      servicePort: 80
```

</TabItem>

</Tabs>

## Configure Plugin on a Route

To configure plugins on a route:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: auth-plugin-config
spec:
  plugins:
    - name: key-auth
      config:
        _meta:
          disable: false
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: get-ip
spec:
  parentRefs:
  - name: apisix
  rules:
  - matches: 
    - path:
        type: Exact
        value: /ip
    filters:
    - type: ExtensionRef
      extensionRef:
        group: apisix.apache.org
        kind: PluginConfig
        name: auth-plugin-config
    backendRefs:
    - name: httpbin
      port: 80
```

</TabItem>

<TabItem value="apisix-crd">

To enable `basic-auth`, `key-auth`, `wolf-rbac`, `jwt-auth`, `ldap-auth`, or `hmac-auth`:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: get-ip
spec:
  ingressClassName: apisix
  http:
    - name: get-ip
      match:
        paths:
          - /ip
    authentication:
      enable: true
      type: keyAuth
    backends:
    - serviceName: httpbin
      servicePort: 80
```

To enable other plugins:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: get-ip
spec:
  ingressClassName: apisix
  http:
    - name: get-ip
      match:
        paths:
          - /ip
      plugins:
      - name: limit-count
        enable: true
        config:
          count: 2
          time_window: 10
          rejected_code: 429
    backends:
    - serviceName: httpbin
      servicePort: 80
```

</TabItem>

</Tabs>

## Configure Global Plugin

To configure a global plugin:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-config
spec:
  plugins:
  - name: clickhouse-logger
    config:
      endpoint_addr: http://clickhouse-clickhouse-installation.apisix.svc.cluster.local:8123
      user: quickstart-user
      password: quickstart-pass
      logtable: test
      database: quickstart_db
```

</TabItem>

<TabItem value="apisix-crd">

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: apisix-global-rule-logging
spec:
  ingressClassName: apisix
  plugins:
  - name: clickhouse-logger
    enable: true
    config:
      endpoint_addr: http://clickhouse-clickhouse-installation.apisix.svc.cluster.local:8123
      user: quickstart-user
      password: quickstart-pass
      logtable: test
      database: quickstart_db
```

</TabItem>

</Tabs>

## Configure Plugin Metadata

To configure plugin metadata:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-config
spec:
  pluginMetadata:
    opentelemetry: {
      "trace_id_source": "x-request-id",
      "resource": {
        "service.name": "APISIX"
      },
      "collector": {
        "address": "simplest-collector:4318",
        "request_timeout": 3,
        "request_headers": {
          "Authorization": "token"
        }
      },
      "batch_span_processor": {
        "drop_on_queue_full": false,
        "max_queue_size": 1024,
        "batch_timeout": 2,
        "inactive_timeout": 1,
        "max_export_batch_size": 16
      },
      "set_ngx_var": true
    }
```

</TabItem>

<TabItem value="apisix-crd">

Not currently supported.

</TabItem>

</Tabs>

## Configure Plugin Config

To create a plugin config and reference it in a route:

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway">

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: example-plugin-config
spec:
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Plugin-Config: "example-response-rewrite"
        X-Plugin-Test: "enabled"
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  rules:
  - matches: 
    - path:
        type: Exact
        value: /ip
    filters:
    - type: ExtensionRef
      extensionRef:
        group: apisix.apache.org
        kind: PluginConfig
        name: example-plugin-config
    backendRefs:
    - name: httpbin
      port: 80
```

</TabItem>

<TabItem value="apisix-crd">

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: example-plugin-config
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Plugin-Config: "example-response-rewrite"
        X-Plugin-Test: "enabled"
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin
spec:
  ingressClassName: apisix
  http:
  - name: get-ip
    match:
      paths:
      - /ip
    backends:
    - serviceName: httpbin
      servicePort: 80
    plugin_config_name: example-plugin-config
```

</TabItem>

</Tabs>

## Configure Gateway Access Information

These configurations allow Ingress Controller users to access the gateway.

<Tabs
groupId="k8s-api"
defaultValue="gateway"
values={[
{label: 'Gateway API', value: 'gateway'},
{label: 'Ingress', value: 'ingress'},
{label: 'APISIX CRD', value: 'apisix-crd'},
]}>

<TabItem value="gateway">

To configure the `statusAddress`:

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-config
spec:
  statusAddress:
    - 10.24.87.13
```

</TabItem>

<TabItem value="ingress">

If you are using Ingress resources, you can configure either `statusAddress` or `publishService`.

To configure the `statusAddress`:

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-config
spec:
  statusAddress:
    - 10.24.87.13
```

To configure the `publishService`:

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-config
spec:
  publishService: apisix-ee-3-gateway-gateway
```

When using `publishService`, make sure your gateway Service is of `LoadBalancer` type the address can be populated. The controller will use the endpoint of this Service to update the status information of the Ingress resource. The format can be either `namespace/svc-name` or simply `svc-name` if the default namespace is correctly set.

</TabItem>

<TabItem value="apisix-crd">

Not supported.

</TabItem>

</Tabs>
