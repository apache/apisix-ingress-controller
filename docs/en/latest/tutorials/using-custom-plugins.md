---
title: Using custom Plugins in APISIX Ingress
keywords:
  - APISIX ingress
  - Apache APISIX
  - Custom Plugins
  - Lua Plugins
description: A tutorial on how you can configure custom Plugins in APISIX Ingress.
---

<head>
    <link rel="canonical" href="https://navendu.me/posts/custom-plugins-in-apisix-ingress/" />
</head>

This tutorial explains how you can configure custom Plugins to work with APISIX Ingress.

## Prerequisites

Before you move on, make sure you:

1. Have access to a Kubernetes cluster. This tutorial uses [minikube](https://github.com/kubernetes/minikube).
2. [Install Helm](https://helm.sh/docs/intro/install/) to deploy the APISIX Ingress controller.

## Deploy httpbin

We will deploy a sample service, [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/), for this tutorial.

You can deploy it to your Kubernetes cluster by running:

```shell
kubectl run httpbin --image kennethreitz/httpbin --port 80
kubectl expose pod httpbin --port 80
```

## Writing a Custom Plugin

In this tutorial we will focus only on configuring custom Plugins to work with APISIX Ingress.

:::tip

To learn more about how to write custom Plugins, see the [documentation](https://apisix.apache.org/docs/apisix/plugin-develop/). You can also write [external Plugins](https://apisix.apache.org/docs/apisix/external-plugin/) in programming languages like Java, Python, and Go.

:::

In this tutorial, we will use a [sample Plugin](https://raw.githubusercontent.com/navendu-pottekkat/apisix-in-kubernetes/master/custom-plugin/plugins/custom-response.lua) that rewrites the response body from the Upstream with a custom value:

```lua {title="custom-response.lua"}
-- some required functionalities are provided by apisix.core
local core = require("apisix.core")

-- define the schema for the Plugin
local schema = {
    type = "object",
    properties = {
        body = {
            description = "custom response to replace the Upstream response with.",
            type = "string"
        },
    },
    required = {"body"},
}

local plugin_name = "custom-response"

-- custom Plugins usually have priority between 1 and 99
-- higher number = higher priority
local _M = {
    version = 0.1,
    priority = 23,
    name = plugin_name,
    schema = schema,
}

-- verify the specification
function _M.check_schema(conf)
    return core.schema.check(schema, conf)
end

-- run the Plugin in the access phase of the OpenResty lifecycle
function _M.access(conf, ctx)
    return 200, conf.body
end

return _M
```

Now we can set up APISIX to utilize this Plugin and enable it for specific Routes.

While one approach is to create a customized build of APISIX that includes the Plugin's code, this is not a simple task.

An alternative method involves generating a [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/) from the Lua code and then mounting it onto the APISIX instance within the Kubernetes environment.

To create the ConfigMap, you can execute the following command:

```shell
kubectl create ns ingress-apisix
kubectl create configmap custom-response-config --from-file=./apisix/plugins/custom-response.lua -n ingress-apisix
```

Now we can deploy APISIX and mount this ConfigMap.

## Deploying APISIX

We will use Helm to deploy APISIX and APISIX Ingress controller.

First, we will update the `values.yaml` file to mount the custom Plugin we created before.

You can configure the Plugin under `customPlugins` as shown below:

```yaml {title="values.yaml"}
customPlugins:
  enabled: true
  plugins:
    - name: "custom-response"
      attrs: {}
      configMap:
        name: "custom-response-config"
        mounts:
          - key: "custom-response.lua"
            path: "/usr/local/apisix/apisix/plugins/custom-response.lua"
```

You should also enable the Plugin by adding it to the `plugins` list:

```yaml {title="values.yaml"}
plugins:
  - api-breaker
  - authz-keycloak
  - basic-auth
  - batch-requests
  - consumer-restriction
  - cors
  ...
  ...
  - custom-response
```

Finally you can enable the Ingress controller and configure the gateway to be exposed to external traffic. For this, set `service.type=NodePort`, `ingress-controller.enabled=true`, and `ingress-controller.config.apisix.serviceNamespace=ingress-apisix` in your `values.yaml` file.

Now we can run `helm install` with this [updated values.yaml](https://raw.githubusercontent.com/navendu-pottekkat/apisix-in-kubernetes/master/custom-plugin/values.yaml) file:

```shell
helm install apisix apisix/apisix -n ingress-apisix --values ./apisix/values.yaml
```

APISIX and APISIX Ingress controller should be ready in some time with the custom Plugin mounted successfully.

## Testing without Enabling the Plugin

First, let's create a Route without our custom Plugin enabled.

We will create a Route using the [ApisixRoute](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_route) CRD:

```yaml {title="route.yaml"}
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: api-route
spec:
  http:
    - name: route
      match:
        hosts:
          - local.navendu.me
        paths:
          - /api
      backends:
        - serviceName: bare-minimum-api
          servicePort: 8080
```

We can now test the created Route:

```shell
curl http://127.0.0.1:52876/api -H 'host:local.navendu.me'
```

This will give back the response from our Upstream service as expected:

```shell
Hello from API v1.0!
```

## Testing the Custom Plugin

Now let's update the Route and enable our custom Plugin on the Route:

```yaml {title="route.yaml"}
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: api-route
spec:
  http:
    - name: route
      match:
        hosts:
          - local.navendu.me
        paths:
          - /api
      backends:
        - serviceName: bare-minimum-api
          servicePort: 8080
      plugins:
        - name: custom-response
          enable: true
          config:
            body: "Hello from your custom Plugin!"
```

Now, our custom Plugin should rewrite the Upstream response with "Hello from your custom Plugin!"

Let's apply this CRD and test the Route and see what happens:

```shell
curl http://127.0.0.1:52876/api -H 'host:local.navendu.me'
```

And as expected, we get the rewritten response from our custom Plugin:

```text {title="output"}
Hello from your custom Plugin!
```
