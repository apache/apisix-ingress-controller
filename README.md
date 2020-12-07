# Apache APISIX for Kubernetes

Use Apache APISIX for Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)

In `apisix-ingress-controller`, all using Custom Resource Definitions (CRDs). Such as configure [plugins](https://github.com/apache/apisix/tree/master/doc/plugins), upport service registration discovery mechanism for upstreams, load balancing and more in Apache APISIX.

Compared with [NGINX Ingress Controller](https://github.com/kubernetes/ingress-nginx), `apisix-ingress-controller` have solved several obvious problems.

* Hot-reload during yaml apply.
* Flexible expansion capabilities.
* Avoid reloading of Ingress itself when updated business services.

`apisix-ingress-controller` belongs to Apache APISIX control plane. Currently it serves for Kubernetes clusters. In the future, we plan to separate the `seven` module to adapt to more deployment modes, such as virtual machine clusters.



## Features

* Declarative configuration for Apache APISIX with Custom Resource Definitions(CRDs), using k8s yaml struct with minimum learning curve.
* Hot-reload during yaml apply.
* Auto register k8s endpoint to upstream(Apache APISIX) node.
* Out of box support for node health check.
* Support load balancing based on pod (upstream nodes).
* Plug-in extension supports hot configuration and immediate effect.
* Ingress controller itself as a plugable hot-reload component.

## Architecture

![Architecture](./docs/images/module-1.png)

### 1.Ingress types

- defines the CRD(CustomResourceDefinition) needed by Apache APISIX

- currently supports ApisixRoute/ApisixService/ApisixUpstream，and other service and route level plugins.

- can be packaged as a stand-alone binary, keep in sync with the ingress definition.

- [CRD design](https://github.com/api7/ingress-controller/issues/3)

### 2. APISIX types

- define interface objects to match concepts from Apache APISIX like route, service, upstream, and plugin.

- can be a packaged as a stand-alone binary, need to match with compatible Apache APISIX version.

- add new types to this module to support new features.

### 3.Seven

- contains main application logic.

- Sync the k8s cluster states to Apache APISIX, based on Apisix-types object.

### 4.Ingress-controller

- driver process for ingress controller, watches k8s apiserver.

- match and covert Apisix-ingress-types to Apisix-types before handing the control over to the above module seven.

Get more [implementation details](./docs/design.md).

## Get started

* [How to install](./docs/install.md)

## Todos

* Support `ApisixSSL` CRD and the logic. #52
* Support retry when synchronization. #59
* Add a complete demo. #9
* Use terratest for testing with Kubernetes. #51
* More todos will display in [issues](https://github.com/apache/apisix-ingress-controller/issues)

## Documents

* [Usage](./docs/usage.md)
* [SDK doc](./docs/develop.md)
* [Design introduction](./docs/design.md)

## FAQ

* [FAQ](./docs/FAQ.md)

## User stories

- [aispeech: Why we create a new k8s ingress controller?(Chinese)](https://mp.weixin.qq.com/s/bmm2ibk2V7-XYneLo9XAPQ)
- [Tencent Cloud: Why choose Apache APISIX to implement the k8s ingress controller?(Chinese)](https://www.upyun.com/opentalk/448.html)

## Contributing

We welcome all kinds of contributions from the open-source community, individuals and partners.

* [Contributing Guide](./docs/contribute.md)

## Community

- Mailing List: Mail to dev-subscribe@apisix.apache.org, follow the reply to subscribe the mailing list.
- QQ Group - 578997126, 552030619
- ![Twitter Follow](https://img.shields.io/twitter/follow/ApacheAPISIX?style=social) - follow and interact with us using hashtag `#ApacheAPISIX`
- [bilibili video](https://space.bilibili.com/551921247)

## License

[Apache License 2.0](https://github.com/api7/ingress-controller/blob/master/LICENSE)
