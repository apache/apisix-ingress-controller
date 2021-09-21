#!/usr/bin/env bash
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

# install utilities
sudo apt-get update
sudo apt-get --yes install apt-transport-https ca-certificates curl gnupg lsb-release make gcc

# install kubectl
sudo curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg
echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
sudo apt-get install kubectl

# Install Golang
GOVERSION=go1.17.1
wget https://golang.org/dl/${GOVERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ${GOVERSION}.linux-amd64.tar.gz
sudo sh -c "echo 'GOBIN=/usr/local/go/bin' >> /etc/profile"
sudo sh -c "echo 'export PATH=\$GOBIN:\$PATH' >> /etc/profile"
source /etc/profile
echo "export GOPATH=\$HOME/go" >> ~/.profile
echo "export PATH=\$GOPATH:\$GOPATH/bin:\$PATH" >> ~/.profile
source ~/.profile

# Install ko
VERSION=0.8.3 # manully choose the latest version, "0.8.3" needs to be manullay modified! Check https://github.com/google/ko/releases for the latest version.
OS=Linux     # or Darwin
ARCH=x86_64  # or arm64, i386, s390x
cd ~ && curl -L https://github.com/google/ko/releases/download/v${VERSION}/ko_${VERSION}_${OS}_${ARCH}.tar.gz | tar xzf - ko
chmod +x ./ko
sudo mv ko /usr/local/bin  # move to system PATH in order to execute `ko` command directly from anywhere. Added by me.

# Install KinD
cd ~ && GO111MODULE="on" go get sigs.k8s.io/kind@v0.11.0
echo "export PATH=\$(go env GOPATH)/bin:\$PATH" >> ~/.profile
source ~/.profile

# Install Helm
curl https://baltocdn.com/helm/signing.asc | sudo apt-key add -
echo "deb https://baltocdn.com/helm/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list
sudo apt-get update
sudo apt-get -y install helm

# Update Helm repo
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add apisix https://charts.apiseven.com
# Use `helm search repo apisix` to search charts about apisix
helm repo update

