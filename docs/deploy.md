# 集群部署

## 安装依赖

通过 [kustomize](https://kustomize.io/) 安装所需要的依赖。

```shell
kubectl kustomize "github.com/apache/apisix-ingress-controller/samples/deploy?ref=master" | kubectl apply -f -
```

上述命令会将 samples/deploy 中声明的配置应用到你的 Kubernetes 集群，如果该目录中的默认配置参数无法满足你的需求，可以考虑修改后再安装：

```shell
kubectl apply -k samples/deploy
```

## 通过CRD定义apisix路由的描述文件

尝试通过ApisixRoute定义一个路由
```
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  name: httpserver-route
spec:
  rules:
  - host: test.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: httpserver
          servicePort: 8080
        path: /hello*
EOF
```
