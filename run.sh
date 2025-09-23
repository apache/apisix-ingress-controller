#!/bin/bash
docker container rm -f $(docker container ps -aq); make kind-up; make install; kind load docker-image --name apisix-ingress-cluster \
  apache/apisix-ingress-controller:dev;  make ginkgo-e2e-test
