# ingress-controller
Apache APISIX ingress for K8s

### 目标

集成k8s，作为一种可选的更好用的ingress。
- 1、保证ingress-controller自身的稳定；
- 2、保证k8s中pod与apisix对象同步；
- 3、支持apisix的扩展特性：route、upstream、consumer、plugin等；


### ingress controller支持特性

- 1、定义apisix CRD(s)，通过yaml定义apisix模型，采用k8s统一的yaml定义方式，可接受程度高；
- 2、yaml文件apply时，支持热变更；
- 3、自动注入k8s endpoint信息到apisix upstream中的node；
- 4、支持node的health check；
- 5、支持在upstream下定义负载均衡；
- 6、支持插件的配置和动态调整；
- 7、ingress controller本身支持热备；


### 设计

![模块划分](https://github.com/iresty/ingress-controller/blob/master/doc/imgs/module.png)

#### 1.Apisix-ingress-types
   - 定义了apisix 在k8s中需要使用的CRD(CustomResourceDefinition)，目前支持ApisixRoute/ApisixService/ApisixUpstream，并且支持service和route级别的plugins定义方式。
   - 该模块可以单独打包，保持与ingress的定义同步；
   - CRD的定义设计参见：https://github.com/iresty/ingress-controller/issues/3

#### 2.Apisix-types
   - 参照apisix api定义了apisix的主要对象route、service、upstream，以及plugin。
   - 该模块可以单独打包，保持与apisix版本一致。
   - 随着功能增强，该模块将会不断完善数据结构的定义；

#### 3.seven
   - 该模块承担了大部分的实现逻辑；
   - 主要职责是基于Apisix-types中定义的对象，将k8s集群中ingress的状态同步到apisix。

#### 4.ingress-controller
   - 我们的ingress controller的启动项目，该模块主要负责watch k8s apiserver
   - 将Apisix-ingress-types对象转换为Apisix-types对象，最后调用seven模块。


### 时序图

![调用流程](https://github.com/iresty/ingress-controller/blob/master/doc/imgs/flow.png)

### 开发文档
[开发文档](doc/dev/develop.md)

### 集群部署
[集群部署](doc/deploy/deploy.md)