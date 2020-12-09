# Installation

## Dependencies

* Kubernetes
* [Deploy Apache APISIX in k8s](https://github.com/apache/apisix/blob/master/kubernetes/README.md)

To install `ingress controller` in k8s, need to care about 3 parts:

1. CRDs: The definitions of Apache APISIX configurations in Kubernetes.

2. [RBAC](https://kubernetes.io/blog/2017/04/rbac-support-in-kubernetes/): This is support by Kubernetes, granting `ingress controller` resource access permissions.

3. Configmap: Contains the necessary configuration for `ingress controller`.

## Kustomize

Install the abovementioned resources by [Kustomize](https://kustomize.io/):

```shell
kubectl kustomize "github.com/apache/apisix-ingress-controller?ref=master" | kubectl apply -f -
```

If the default parameters in samples/deploy are not good for you, just tweak them and run:

```shell
kubectl apply -k /path/to/apisix-ingress-controller/samples/deploy
```

## Helm

// todo

## Master branch builds

```shell
docker build -t apache/ingress-controller:v0.1.0 ../.
```

## Next

* [Usage](./usage.md)
