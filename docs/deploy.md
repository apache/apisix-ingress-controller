# 集群部署

## 配置RBAC
示例文件在项目下的samples下
* 执行下面的命令之前按需修改一下name和namespaces
### 1.创建ServiceAccount

```shell
kubectl apply -f samples/deploy/rbac/service_account.yaml
```

### 2.创建ClusterRole
```shell
kubectl apply -f samples/deploy/rbac/apisix_view_clusterrole.yaml
```

### 3.创建ClusterRoleBinding
```shell
kubectl apply -f samples/deploy/rbac/apisix_view_clusterrolebinding.yaml
```

## 配置 ConfigMap
* 执行命令之前按需修改

```shell
kubectl apply -f samples/deploy/configmap/cloud.yaml
```

## 配置 Deployment
* 执行命令之前按需修改name和namespaces，并且补充APISIX ingress controller image路径和版本号


```shell
kubectl apply -f samples/deploy/deployment/ingress-controller.yaml
```

## CRD定义文件

如果APISIX CRD还未定义，执行以下脚本
```
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

## 通过CRD定义apisix路由的描述文件

尝试通过ApisixRoute定义一个路由
```
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  name: httpserver-route
  namespace: cloud
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
