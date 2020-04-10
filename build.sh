#!/bin/bash

pwd=`pwd`

sed -i -e "s%#etcdAddress#%`echo $ETCD_SERVER_INTERNAL`%g" ${pwd}/conf.json
sed -i -e "s%#k8sAuthFile#%`echo $K8S_AUTH_FILE`%g" ${pwd}/conf.json
sed -i -e "s%#syslogAddress#%`echo $SYSLOG_HOST`%g" ${pwd}/conf.json
sed -i -e "s%#apisixBaseUrl#%`echo $APISIX_BASE_URL`%g" ${pwd}/conf.json

cd /root/ingress-controller
exec ./ingress-controller

