# Usage of Ingress controller

In this article, we will use ingress controller CRDs (CustomResourceDefinition) to define routing rules against the admin api of Apache APISIX.

## Scenes

Configure a simple routing rule through the ingress controller CRDs. After synchronizing to the gateway, the data traffic is accessed to the back-end service through Apache APISIX. Then, we gradually add or remove plug-ins to the routing to achieve functional expansion.

As shown below.

![scene](./images/scene.png)

## A simple example

Define the simplest route to direct traffic to the back-end service, the back-end service is named `httpserver`.

As shown below.

![first](./images/first.png)

Now we define with CRDs as follows.

1. Define Upstream with `ApisixUpstream`

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: foo
  namespace: cloud
spec:
  ports:
  - port: 8080
    loadbalancer:
      type: chash
      hashOn: header
      key: hello
```

2. Define Service with `ApisixService`

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixService
metadata:
  name: foo
  namespace: cloud
spec:
  upstream: foo
  port: 8080
```

3. Define Route with `ApisixRoute`

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  name: foo-route
  namespace: cloud
spec:
  rules:
  - host: test.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: foo
          servicePort: 8080
        path: /hello*
```

Tips: When defining `ApisixUpstream`, there is no need to define a specific pod ip list, the ingress controller will do service discovery based on namespace/name/port composite index.

List the way to define the above rules using `Admin API` to facilitate comparison and understanding.

```shell
# 1. Define upstream: foo-upstream id=1 (the ip list in nodes is an example, no specific meaning)
curl -XPUT http://127.0.0.1:9080/apisix/admin/upstreams/1 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -d '
{
    "nodes": {
        "10.244.143.48:8080": 100,
        "10.244.102.43:8080": 100,
        "10.244.102.63:8080": 100
    },
    "desc": "foo-upstream",
    "type": "roundrobin"
}
'
# 2. Define service: foo-service, id=2, binding upstream: foo-upstream
curl -XPUT http://127.0.0.1:9080/apisix/admin/services/2 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -d '
{
    "desc": "foo-service",
    "upstream_id": 1
}
'

# 3. Define route: foo-route， id=3， binding service: foo-service

curl -XPUT http://127.0.0.1:9080/apisix/admin/routes/3 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -d '
{
    "desc": "foo-route",
    "uri": "/hello*",
    "host": "test.apisix.apache.org",
    "service_id": "2"
}'
```

## Add a plugin

Next, take the `proxy-rewrite` plugin as an example.

Add plug-ins through admin api to achieve the purpose of rewriting upstream uri.

e.g. test.apisix.apache.org/hello -> test-rewrite.apisix.apache.org/copy/hello

With CRDs, use `ApisixService` as example.

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixService
metadata:
  name: foo
  namespace: cloud
spec:
  upstream: foo
  port: 8080
  plugins:
  - enable: true
    name: proxy-rewrite
    config:
    regex_uri:
    - '^/(.*)'
    - '/copy/$1'
    scheme: http
    host: test-rewrite.apisix.apache.org
```

For facilitating understanding, show the way with `Admin API` as below.

```shell
curl http://127.0.0.1:9080/apisix/admin/services/2 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -X PUT -d '
{
    "desc": "foo-service",
    "upstream_id": "1",
    "plugins": {
        "proxy-rewrite": {
            "regex_uri": ["^/(.*)", "/copy/$1"],
            "scheme": "http",
            "host": "test-rewrite.apisix.apache.org"
        }
    }
}'
```

It can be found that the way of defining plugins is almost the same, except that the format is changed from `json` to `yaml`.

By defining the plug-in in CRDs, you can disable the plug-in by setting `enable: false` without deleting it. Keep the original configuration for easy opening next time.

Tips: ApisixRoute and ApisixService both support plugins definition.
