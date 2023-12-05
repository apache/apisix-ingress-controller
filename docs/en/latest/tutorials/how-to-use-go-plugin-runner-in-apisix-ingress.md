---
title: How to use go-plugin-runner with APISIX Ingress
keywords:
  - Apache APISIX Ingress
  - Plugin
  - Ingress Controller
  - Go Plugin Runner
  - multi language
description: This document walks through how you can use the go plugin runner in the APISIX ingress controller
---

<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

## Description

Based on version 0.3 of the go-plugin-runner plugin and version 1.4.0 of APISIX Ingress, this document walks through how you can use the go plugin runner in the APISIX ingress controller. This article goes through steps as follows:

1. Prepare the environment.
2. Create the cluster.
3. Build a container image that includes the go-plugin-runner.
4. Customize the Helm chart package.
5. Install and deploy.
6. Verify the function.

It is guaranteed that the final result can be derived in full based on this environment example as follows:

```bash
go-plugin-runner: 0.3
APISIX Ingress: 1.4.0
kind: v0.12.0
kubectl version(Client/Server): v1.23.5/v1.23.4
golang: 1.18
```

## Build a cluster environment

Select `kind` to build a local cluster environment. The command is as follows:

```bash
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF
```

## Build the go-plugin-runner executable

Choose a folder address `/home/chever/api7/cloud_native/tasks/plugin-runner` and place our `apisix-go-plugin-runner` project in this folder. Then you need to go to the `apisix-go-plugin-runner/cmd/go-runner/plugins` directory and write the plugins you need in that directory.

After writing the plugins, start compiling the executable formally, and note here that you should build **static executables**, not dynamic ones.

The package compile command is as follows.

```bash
CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' .
```

This successfully packages a statically compiled `go-runner` executable in the `apisix-go-plugin-runner/cmd/go-runner/` directory.

## Build Docker Image

The image is built here in preparation for installing APISIX later using `helm`.

### Write Dockerfile

Return to the path `/home/chever/api7/cloud_native/tasks/plugin-runner` and create a Dockerfile in that directory, a demonstration of which is given here.

```dockerfile
# DockerfileForRunner
FROM apache/apisix:2.13.1-alpine

COPY ./apisix-go-plugin-runner /usr/local/apisix-go-plugin-runner
```

Here I will again emphasize the path address as follows where the executable file is located.

```bash
/usr/local/apisix-go-plugin-runner/cmd/go-runner/go-runner
```

Please make a note of this address. We will use it in the rest of the configuration.

### Begin to build Docker Image

Start building a Docker image based on the Dockerfile. The command is executed in the `/home/chever/api7/cloud_native/tasks/plugin-runner` directory. The command is as follows:

```bash
docker build -t apisix/forrunner:0.1 .
```

Command Explanation: Build an image with the name `apisix/forrunner` and mark it as version 0.1.

### Load the image to the cluster environment

```bash
kind load docker-image apisix/forrunner:0.1
```

Load the image into the kind cluster environment to pull the custom local image for installation during the helm installation.

## Install APISIX Ingress

Then install APISIX using helm with the following command in the directory of Apache APISIX Helm Chart:

```bash
#  We use Apisix 3.0 in this example. If you're using Apisix v2.x, please set to v2
ADMIN_API_VERSION=v3
helm install apisix apisix/apisix \
  --set service.type=NodePort \
  --set apisix.image.repository=custom/apisix \
  --set apisix.image.tag=v0.1 \
  --set extPlugin.enabled=true \
  --set extPlugin.cmd='{"/usr/local/apisix-go-plugin-runner/go-runner", "run"}' \
  --set ingress-controller.enabled=true \
  --create-namespace \
  --namespace apisix-admin \
  --set ingress-controller.config.apisix.serviceName=apisix-admin \
  --set ingress-controller.config.apisix.adminAPIVersion=$ADMIN_API_VERSION
```

## Create httpbin service and ApisixRoute resources

Create an httpbin backend resource to run with the deployed ApisixRoute resource to test that the functionality is working correctly.

### Create httpbin service

Create an httpbin service with the following command:

```bash
kubectl run httpbin --image kennethreitz/httpbin --port 80
```

Expose the port with the following command:

```bash
kubectl expose pod httpbin --port 80
```

### Create ApisixRoute Resource

Create the `go-plugin-runner-route.yaml` file to enable the ApisixRoute resource, with the following configuration file:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: plugin-runner-demo
spec:
  http:
  - name: rule1
    match:
      hosts:
      - local.httpbin.org
      paths:
      - /get
    backends:
    - serviceName: httpbin
      servicePort: 80
    plugins:
    - name: ext-plugin-pre-req
      enable: true
      config:
        conf:
        - name: "say"
          value: "{\"body\": \"hello\"}"
```

The create resource command is as follows:

```bash
kubectl apply -f go-plugin-runner-route.yaml
```

## Test

The command is as follows to test if the plugin written in Golang is working correctly:

```bash
kubectl exec -it -n ${namespace of Apache APISIX} ${Pod name of Apache APISIX} -- curl http://127.0.0.1:9080/get -H 'Host: local.httpbin.org'
```

And you will see the result as follows:

```bash
Defaulted container "apisix" out of: apisix, wait-etcd (init)
hello
```
