---
title: ApisixPluginConfig
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixPluginConfig
description: Guide to using ApisixPluginConfig custom Kubernetes resource.
---

`ApisixPluginConfig` is a Kubernetes CRD that can be used to extract commonly used Plugins and can be bound directly to multiple Routes.

See [reference](https://apisix.apache.org/docs/ingress-controller/references/apisix_pluginconfig_v2) for the full API documentation.

The example below shows how you can configure an `ApisixPluginConfig` resource:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: echo-and-cors-apc
spec:
  plugins:
  - name: echo
    enable: true
    config:
      before_body: "This is the prologue"
      after_body: "This is the epilogue"
      headers:
       X-Foo: v1
       X-Foo2: v2
  - name: cors
    enable: true
```

You can then configure a Route to use the `echo-and-cors-apc` Plugin configuration:

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
      weight: 10
    plugin_config_name: echo-and-cors-apc
```
