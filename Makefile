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
default: help

VERSION ?= 0.2.0
RELEASE_SRC = apache-apisix-ingress-controller-${VERSION}-src
IMAGE_TAG ?= "dev"

GINKGO ?= $(shell which ginkgo)
GITSHA ?= $(shell git rev-parse --short=7 HEAD)
OSNAME ?= $(shell uname -s | tr A-Z a-z)
OSARCH ?= $(shell uname -m | tr A-Z a-z)
PWD ?= $(shell pwd)
ifeq ($(OSARCH), x86_64)
	OSARCH = amd64
endif

VERSYM="github.com/api7/ingress-controller/pkg/version._buildVersion"
GITSHASYM="github.com/api7/ingress-controller/pkg/version._buildGitRevision"
BUILDOSSYM="github.com/api7/ingress-controller/pkg/version._buildOS"
GO_LDFLAGS ?= "-X=$(VERSYM)=$(VERSION) -X=$(GITSHASYM)=$(GITSHA) -X=$(BUILDOSSYM)=$(OSNAME)/$(OSARCH)"
E2E_CONCURRENCY ?= 1
E2E_SKIP_BUILD ?= 0

### build:            Build apisix-ingress-controller
build:
	go build \
		-o apisix-ingress-controller \
		-ldflags $(GO_LDFLAGS) \
		main.go

### build-image:      Build apisix-ingress-controller image
build-image:
	docker build -t apache/apisix-ingress-controller:$(IMAGE_TAG) .

### lint:             Do static lint check
lint:
	golangci-lint run

### gofmt:            Format all go codes
gofmt:
	find . -type f -name "*.go" | xargs gofmt -w -s

### unit-test:        Run unit test cases
unit-test:
	go test -cover -coverprofile=coverage.txt ./...

### e2e-test:         Run e2e test cases (minikube is required)
e2e-test: ginkgo-check build-image-to-minikube
	kubectl apply -f $(PWD)/samples/deploy/crd/v1beta1/ApisixRoute.yaml
	kubectl apply -f $(PWD)/samples/deploy/crd/v1beta1/ApisixUpstream.yaml
	kubectl apply -f $(PWD)/samples/deploy/crd/v1beta1/ApisixService.yaml
	kubectl apply -f $(PWD)/samples/deploy/crd/v1beta1/ApisixTls.yaml
	cd test/e2e && ginkgo -cover -coverprofile=coverage.txt -r --randomizeSuites --randomizeAllSpecs --trace -p --nodes=$(E2E_CONCURRENCY)

.PHONY: ginkgo-check
ginkgo-check:
ifeq ("$(wildcard $(GINKGO))", "")
	@echo "ERROR: Need to install ginkgo first, run: go get -u github.com/onsi/ginkgo/ginkgo"
	exit 1
endif

# build images to minikube node directly, it's an internal directive, so don't
# expose it's help message.
build-image-to-minikube:
ifeq ($(E2E_SKIP_BUILD), 0)
	@minikube version > /dev/null 2>&1 || (echo "ERROR: minikube is required."; exit 1)
	@eval $$(minikube docker-env);\
	docker build -t apache/apisix-ingress-controller:$(IMAGE_TAG) .
endif

### license-check:    Do Apache License Header check
license-check:
ifeq ("$(wildcard .actions/openwhisk-utilities/scancode/scanCode.py)", "")
	git clone https://github.com/apache/openwhisk-utilities.git .actions/openwhisk-utilities
	cp .actions/ASF* .actions/openwhisk-utilities/scancode/
endif
	.actions/openwhisk-utilities/scancode/scanCode.py --config .actions/ASF-Release.cfg ./

### help:             Show Makefile rules
help:
	@echo Makefile rules:
	@echo
	@grep -E '^### [-A-Za-z0-9_]+:' Makefile | sed 's/###/   /'

### release-src:      Release source
release-src:
	tar -zcvf $(RELEASE_SRC).tgz \
	--exclude .github \
	--exclude .git \
	--exclude .idea \
	--exclude .gitignore \
	--exclude .DS_Store \
	--exclude docs \
	--exclude samples \
	--exclude test \
	--exclude release \
	--exclude $(RELEASE_SRC).tgz \
	.

	gpg --batch --yes --armor --detach-sig $(RELEASE_SRC).tgz
	shasum -a 512 $(RELEASE_SRC).tgz > $(RELEASE_SRC).tgz.sha512

	mkdir -p release
	mv $(RELEASE_SRC).tgz release/$(RELEASE_SRC).tgz
	mv $(RELEASE_SRC).tgz.asc release/$(RELEASE_SRC).tgz.asc
	mv $(RELEASE_SRC).tgz.sha512 release/$(RELEASE_SRC).tgz.sha512

.PHONY: build lint help

