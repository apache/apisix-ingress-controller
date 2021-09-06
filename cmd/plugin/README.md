# apisix-ingress-controller plugin

The apisix-ingress-controoler plugin can quikly show the apisix routes and upstreams in kubernetes cluster.

# Quikly start 

```shell
git clone http://github.com/apache/apisix-ingress-controller 
cd cmd/plugin
sh install-plugin.sh
```

```shell 
# Show your routes in kubernetes.
kubectl apisix-ingress-controller routes -n <APISIX_INGRESS_NS>
# Show your upstreams in kubernetes.
kubectl apisix-ingress-controller  upstreams -n <APISIX_INGRESS_NS>
# Show one upstream information's 
kubectl apisix-ingress-controller upstreams --upstream-id <UPSTREAM_ID> -n <APISIX_INGRESS_NS>

# get more help information's 
kubectl apisix-ingress-controller --help
```