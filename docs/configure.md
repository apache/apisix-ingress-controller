# Configure

The APISIX Ingress Controller is a Kubernetes Ingress Controller that implements the Gateway API. This document describes how to configure the APISIX Ingress Controller.

## Example

```yaml
log_level: "info"                               # The log level of the APISIX Ingress Controller.
                                                # the default value is "info".

controller_name: apisix.apache.org/apisix-ingress-controller  # The controller name of the APISIX Ingress Controller,
                                                          # which is used to identify the controller in the GatewayClass.
                                                          # The default value is "apisix.apache.org/apisix-ingress-controller".

leader_election_id: "apisix-ingress-controller-leader" # The leader election ID for the APISIX Ingress Controller.
                                                        # The default value is "apisix-ingress-controller-leader".
```

### Controller Name

The `controller_name` field is used to identify the `controllerName` in the GatewayClass.

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: apisix
spec:
  controllerName: "apisix.apache.org/apisix-ingress-controller"
```
