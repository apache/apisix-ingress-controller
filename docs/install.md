# Installation

## Dependencies

* Kubernetes
* [Deploy Apache APISIX in k8s](https://github.com/apache/apisix/blob/master/kubernetes/README.md)

To install `ingress controller` in k8s, need to care about 3 parts:

1. CRDs: The definitions of Apache APISIX configurations in Kubernetes.

2. [RBAC](https://kubernetes.io/blog/2017/04/rbac-support-in-kubernetes/): This is support by Kubernetes, granting `ingress controller` resource access permissions.

3. Configmap: Contains the necessary configuration for `ingress controller`.

## CRDs installation

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

## RBAC configuration

* Create ServiceAccount

```shell
kubectl apply -f ../samples/deploy/rbac/service_account.yaml
```

* Create ClusterRole

```shell
kubectl apply -f ../samples/deploy/rbac/apisix_view_clusterrole.yaml
```

* Create ClusterRoleBinding

```shell
kubectl apply -f ../samples/deploy/rbac/apisix_view_clusterrolebinding.yaml
```

## Configmap for ingress controller

Pay attention to the `namespace` and `APISIX address` in configmap.

```shell
kubectl apply -f ../samples/deploy/configmap/cloud.yaml
```

## Deploy ingress controller

[How to build image from master branch?](#Master-branch-builds)

```shell
kubectl apply -f ../samples/deploy/deployment/ingress-controller.yaml
```

## Helm

// todo

## Master branch builds

```shell
docker build -t apache/ingress-controller:v0.1.0 ../.
```

## Next

* [Usage](./usage.md)
