# ingress-controller
ingress controller for K8s

### 目标
```
1、保证ingress-controller自身的稳定；
2、保证k8s中pod与apisix对象同步；
3、支持apisix的扩展特性：route、upstream、consumer、plugin等；
```

### ingress controller支持特性

```
1、将apisix中的对象定义为一个或者多个k8s CRD(s)，通过yaml定义apisix模型；
2、yaml文件apply时，支持热变更；
3、支持将k8s pod信息同步到upstream中的node；
4、支持node的health check；
5、支持在upstream下定义负载均衡；
6、支持插件的配置和动态调整；
7、ingress controller本身支持热备；
```

### 调用流程

![调用流程](https://github.com/iresty/ingress-controller/blob/master/doc/imgs/flow.png)
