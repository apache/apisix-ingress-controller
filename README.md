# Apache APISIX for Kubernetes

Use Apache APISIX for Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)

Configure [plugins](https://github.com/apache/apisix/tree/master/doc/plugins), load balancing and more in Apache APISIX for Kubernetes Services, support service registration discovery mechanism for upstreams. All using Custom Resource Definitions (CRDs).

## Features

* Declarative configuration for Apache APISIX with Custom Resource Definitions(CRDs), using k8s yaml struct with minimum learning curve.
* Hot-reload during yaml apply.
* Auto register k8s endpoint to upstream(Apache APISIX) node.
* Out of box support for node health check.
* Support load balancing based on pod (upstream nodes).
* Plug-in extension supports hot configuration and immediate effect.
* Ingress controller itself as a plugable hot-reload component.

## Get started

### Dependencies

* Kubernetes
* [Deploy Apache APISIX in k8s](https://github.com/apache/apisix/blob/master/kubernetes/README.md)

To install `ingress controller` in k8s, need to care about 3 parts:

1. CRDs: They are the data structure of Apache APISIX in Kubernetes, used to define route, service, upstream, plugins, etc.

2. RBAC: This is a function of Kubernetes, granting `ingress controller` resource access permissions.

3. Configmap: Contains the necessary configuration for `ingress controller`.

### CRDs installation

Install CRDs in Kubernetes

```shell
kubectl apply -f - <<EOF
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apisixroutes.apisix.apache.org
spec:
  group: apisix.apache.org
  versions:
    - name: v1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: apisixroutes
    singular: apisixroute
    kind: ApisixRoute
    shortNames:
    - ar

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apisixservices.apisix.apache.org
spec:
  group: apisix.apache.org
  versions:
    - name: v1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: apisixservices
    singular: apisixservice
    kind: ApisixService
    shortNames:
    - as

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apisixupstreams.apisix.apache.org
spec:
  group: apisix.apache.org
  versions:
    - name: v1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: apisixupstreams
    singular: apisixupstream
    kind: ApisixUpstream
    shortNames:
    - au

EOF
```

### RBAC configuration

* Create ServiceAccount

```shell
kubectl apply -f samples/deploy/rbac/service_account.yaml
```

* Create ClusterRole

```shell
kubectl apply -f samples/deploy/rbac/apisix_view_clusterrole.yaml
```

* Create ClusterRoleBinding

```shell
kubectl apply -f samples/deploy/rbac/apisix_view_clusterrolebinding.yaml
```

### Configmap for ingress controller

Pay attention to the `namespace` and `APISIX address` in configmap.

```shell
kubectl apply -f samples/deploy/configmap/cloud.yaml
```

### Deploy ingress controller

[How to build image from master branch?](# Master branch builds)

```shell
kubectl apply -f samples/deploy/deployment/ingress-controller.yaml
```

### Helm

// todo

## Document

* [SDK doc](./doc/dev/develop.md)
* [Design introduction](./doc/design/design.md)

## Master branch builds

```shell
docker build -t apache/ingress-controller:v0.1.0 .
```

## Seeking help

- Mailing List: Mail to dev-subscribe@apisix.apache.org, follow the reply to subscribe the mailing list.
- QQ Group - 578997126, 552030619
- [Slack Workspace](http://s.apache.org/slack-invite) - join `#apisix` on our Slack to meet the team and ask questions
- ![Twitter Follow](https://img.shields.io/twitter/follow/ApacheAPISIX?style=social) - follow and interact with us using hashtag `#ApacheAPISIX`
- [bilibili video](https://space.bilibili.com/551921247)

## License

[Apache License 2.0](https://github.com/api7/ingress-controller/blob/master/LICENSE)

