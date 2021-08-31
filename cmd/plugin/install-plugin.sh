#!/bin/bash

go build -o kubectl-apisix_ingress_controller .

cp $(PWD)/kubectl-apisix_ingress_controller /usr/local/bin/
chmod +x /usr/local/bin/kubectl-apisix_ingress_controller
