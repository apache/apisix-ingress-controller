# 开发文档
## 依赖
#### 1.K8s环境
#### 2.apisix server
#### 3.golang >= 1.13 (go mod)

## 开发环境搭建
#### 1. k8s环境最小安装
安装minikube，参考https://kubernetes.io/docs/tasks/tools/install-minikube/

生产和测试环境建议使用k8s集群部署方式。

#### 2. 安装apisix
推荐 在k8s集群内部署apisix

如果你需要同时修改apisix和ingress-controller，也可以选择在集群外部署apisix。

###### 注意：ingress-controller会向apisix注册服务副本IP，若在集群外部部署apisix，apisix将不能访问k8s内部的upstream

## 开发环境配置
#### 本地配置kube config文件，方便本地调试
1.启动minikube；

2.config文件位置：~/.kube/config

3.将config文件copy一份到本地开发环境，并将文件路径配置到 $GOPATH/src/github.com/api7/ingress-controller/conf/conf.json中的k8sAuth配置项;

#### 配置apisix服务地址
不管选择哪种部署方式，请将apisix的服务地址配置到 $GOPATH/src/github.com/api7/ingress-controller/conf/conf.json中，配置项为conf/apisix/base_url

## 本地启动ingress-controller

###### 推荐使用IDE，比如vscode或者goland。

1.在k8s中创建CRD定义文件；
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

2.参考上面的说明，将conf.json配置好。

3.在ingress-controller根目录执行：
```
# go run main.go -logtostderr -v=5
```

4.此时程序可能会打印一些错误日志，提示找不到资源，继续执行以下步骤，通过CRD定义路由
以apisix反向代理httpserver服务为例（你可以选择任意测试项目部署），基本格式如下，可以copy执行
##### 定义 ApisixRoute
事实上，为了减少ingress迁移带来的麻烦，我们在ApisixRoute的结构上尽量保持与原生ingress一致。

    与nginx-ingress的配置差异点在：
    1.apiVersion 和 kind 不同；
    2.path加了通配符； 例如：path: /hello*

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
另外，ApisixRoute在满足快速从ingress迁移的基础上，还在不断的完善annotation的定义，以及对插件的增强。你也可以这样写
```
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  annotations:
    k8s.apisix.apache.org/cors-allow-headers: DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization,openID,audiotoken
    k8s.apisix.apache.org/cors-allow-methods: HEAD,GET,POST,PUT,PATCH,DELETE
    k8s.apisix.apache.org/cors-allow-origin: '*'
    k8s.apisix.apache.org/enable-cors: "true"
    k8s.apisix.apache.org/ssl-redirect: "false"
    k8s.apisix.apache.org/whitelist-source-range: 1.2.3.4,2.2.0.0/16
  name: httpserver-route
spec:
  rules:
  - host: test1.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: httpserver
          servicePort: 8080
        path: /hello*
        plugins:
        - enable: true
          name: proxy-rewrite
          config:
            regex_uri:
            - '^/(.*)'
            - '/voice-copy-outer-service/$1'
            scheme: http
            host: internalalpha.talkinggenie.com
            enable_websocket: true
```
config中定义需要按照 插件```proxy-rewrite```的schema定义；

如果插件的schema是一个数组，需要用将config字段修改为config_set；


##### 定义 ApisixService
```
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v1
kind: ApisixService                   # apisix service
metadata:
  name: httpserver
spec:
  upstream: httpserver          # upstream = default/httpserver (namespace/upstreamName)
  port: 8080                        # 在service上定义端口号
  plugins:
  - name: aispeech-chash
    enable: true
    config:
      uri_args:
        - "userId"
        - "productId|deviceName"
      key: "apisix-chash-key"
EOF
```
ApisixService对插件的支持类似ApisixRoute

##### 定义 ApisixUpstream
```
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream                  # apisix upstream
metadata:
  name: httpserver      # default/httpserver
spec:
  ports:
  - port: 8080
    loadbalancer:
      type: chash
      hashOn: header
      key: hello
EOF
```
现在，你可以按照CRD的格式，尝试修改这些yaml，看看会不会同步到apisix。

Enjoy！如果有任何问题，欢迎反馈issue。
