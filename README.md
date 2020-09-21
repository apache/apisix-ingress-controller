# ingress-controller
Apache APISIX ingress for K8s

### Goal

A k8s compatible ingress controller.
1. stability；
2. stay in sync with Apache APISIX pod in k8s cluster；
3. Full Apache APISIX extension support：route, upstream, consumer, plugin, etc；


### ingress controller features

 1. add Apache APISIX Custom Resource Definition(s) via k8s yaml syntax(with minimum learning curve);
 2. hot-reload during yaml apply;
 3. auto register k8s endpoint to upstream(Apache APISIX) node;
 4. out of box support for node health check；
 5. support upstream node defined load balancing；
 6. extension plugin config hot-reload and dynamic tuning；
 7. ingress controller itself as a plugable hot-reload component；


### Design

![Architecture](https://github.com/iresty/ingress-controller/blob/master/doc/imgs/module.png)

#### 1.Apisix-ingress-types
   - defines the CRD(CustomResourceDefinition) needed by Apache APISIX
   - currently supports ApisixRoute/ApisixService/ApisixUpstream，and other service and route level plugins;
   - can be packaged as a stand-alone binary, keep in sync with the ingress definition;
   - CRD design see：https://github.com/iresty/ingress-controller/issues/3;

#### 2.Apisix-types
   - define interface objects to match concepts from Apache APISIX like route, service, upstream, and plugin;
   - can be a packaged as a stand-alone binary, need to match with compatible Apache APISIX version;
   - add new types to this module to support new features;

#### 3.seven
   - contains main application logic;
   - Sync the k8s cluster states to Apache APISIX, based on Apisix-types object;

#### 4.ingress-controller
   - driver process for ingress controller; watches k8s apiserver;
   - match and covert Apisix-ingress-types to Apisix-types before handing the control over to the above module seven;


### Sequence Diagram

![Sequence Diagram](https://github.com/iresty/ingress-controller/blob/master/doc/imgs/flow.png)

### Documentation
[SDK Doc](doc/dev/develop.md)

### Deployment
[Deployment](doc/deploy/deploy.md)
