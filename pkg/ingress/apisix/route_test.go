package apisix

import "testing"

func TestApisixRoute_Convert(t *testing.T) {

}

var routeYaml = `
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  annotations:
    k8s.apisix.apache.org/cors-allow-headers: DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization,openID,audiotoken
    k8s.apisix.apache.org/cors-allow-methods: HEAD,GET,POST,PUT,PATCH,DELETE
    k8s.apisix.apache.org/cors-allow-origin: '*'
    k8s.apisix.apache.org/enable-cors: "true"
    k8s.apisix.apache.org/ssl-redirect: "false"
    k8s.apisix.apache.org/whitelist-source-range: 58.210.212.110,10.244.0.0/16
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apisix.apache.org/v1","kind":"ApisixRoute","metadata":{"annotations":{},"name":"httpserver-route","namespace":"cloud"},"spec":{"rules":[{"host":"test1.apisix.apache.org","http":{"paths":[{"backend":{"serviceName":"httpserver","servicePort":8080},"path":"/hello*"},{"backend":{"serviceName":"api6","servicePort":80},"path":"/api6"}]}}]}}
  creationTimestamp: "2020-02-11T03:00:55Z"
  generation: 32
  name: httpserver-route
  namespace: cloud
  resourceVersion: "9808944"
  selfLink: /apis/apisix.apache.org/v1/namespaces/cloud/apisixroutes/httpserver-route
  uid: b8ce6dd6-4c7a-11ea-9952-080027b01891
spec:
  rules:
  - host: test1.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: api6
          servicePort: 80
        path: /test*
        plugins:
        - config:
            key: apisix-chash-key
            uri_args:
            - pId
            - userId|device
          enable: true
          name: aispeech-chash
      - backend:
          serviceName: httpserver
          servicePort: 8080
        path: /hello*
        plugins:
        - config:
            key: apisix-chash-key
            uri_args:
            - productId2
            - productId|deviceName
          enable: true
          name: aispeech-chash
`