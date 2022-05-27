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

VERSION ?= 1.4.1
RELEASE_SRC = apache-apisix-ingress-controller-${VERSION}-src
REGISTRY ?="localhost:5000"
IMAGE_TAG ?= dev
ENABLE_PROXY ?= true

GITSHA ?= "no-git-module"
ifneq ("$(wildcard .git)", "")
	GITSHA = $(shell git rev-parse --short=7 HEAD)
endif
GINKGO ?= $(shell which ginkgo)
OSNAME ?= $(shell uname -s | tr A-Z a-z)
OSARCH ?= $(shell uname -m | tr A-Z a-z)
PWD ?= $(shell pwd)
ifeq ($(OSARCH), x86_64)
	OSARCH = amd64
endif

VERSYM="github.com/apache/apisix-ingress-controller/pkg/version._buildVersion"
GITSHASYM="github.com/apache/apisix-ingress-controller/pkg/version._buildGitRevision"
BUILDOSSYM="github.com/apache/apisix-ingress-controller/pkg/version._buildOS"
GO_LDFLAGS ?= "-X=$(VERSYM)=$(VERSION) -X=$(GITSHASYM)=$(GITSHA) -X=$(BUILDOSSYM)=$(OSNAME)/$(OSARCH)"
E2E_CONCURRENCY ?= 1
E2E_SKIP_BUILD ?= 0

### build:                Build apisix-ingress-controller
.PHONY: build
build:
	go build \
		-o apisix-ingress-controller \
		-ldflags $(GO_LDFLAGS) \
		main.go

### build-image:          Build apisix-ingress-controller image
.PHONY: build-image
build-image:
	docker build -t apache/apisix-ingress-controller:$(IMAGE_TAG) --build-arg ENABLE_PROXY=true .

### lint:                 Do static lint check
.PHONY: lint
lint:
	golangci-lint run

### unit-test:            Run unit test cases
.PHONY: unit-test
unit-test:
	go test -cover -coverprofile=coverage.txt ./...

### e2e-test:             Run e2e test cases (in existing clusters directly)
.PHONY: e2e-test
e2e-test: ginkgo-check push-images
	kubectl apply -k $(PWD)/samples/deploy/crd
	cd test/e2e \
		&& go mod download \
		&& export REGISTRY=$(REGISTRY) \
		&& ACK_GINKGO_RC=true ginkgo -cover -coverprofile=coverage.txt -r --randomizeSuites --randomizeAllSpecs --trace --nodes=$(E2E_CONCURRENCY) --focus=$(E2E_FOCUS)

### e2e-test-local:        Run e2e test cases (kind is required)
.PHONY: e2e-test-local
e2e-test-local: kind-up e2e-test

.PHONY: ginkgo-check
ginkgo-check:
ifeq ("$(wildcard $(GINKGO))", "")
	@echo "ERROR: Need to install ginkgo first, run: go get -u github.com/onsi/ginkgo/ginkgo"
	exit 1
endif


### push-ingress-images:  Build and push Ingress image used in e2e test suites to kind or custom registry.
.PHONY: push-ingress-images
push-ingress-images:
	docker build -t apache/apisix-ingress-controller:$(IMAGE_TAG) --build-arg ENABLE_PROXY=$(ENABLE_PROXY) .
	docker tag apache/apisix-ingress-controller:$(IMAGE_TAG) $(REGISTRY)/apache/apisix-ingress-controller:$(IMAGE_TAG)
	docker push $(REGISTRY)/apache/apisix-ingress-controller:$(IMAGE_TAG)

### push-images:  Push images used in e2e test suites to kind or custom registry.
.PHONY: push-images
push-images:
ifeq ($(E2E_SKIP_BUILD), 0)
	docker pull apache/apisix:2.13.1-alpine
	docker tag apache/apisix:2.13.1-alpine $(REGISTRY)/apache/apisix:dev
	docker push $(REGISTRY)/apache/apisix:dev

	docker pull bitnami/etcd:3.4.14-debian-10-r0
	docker tag bitnami/etcd:3.4.14-debian-10-r0 $(REGISTRY)/bitnami/etcd:3.4.14-debian-10-r0
	docker push $(REGISTRY)/bitnami/etcd:3.4.14-debian-10-r0

	docker pull kennethreitz/httpbin
	docker tag kennethreitz/httpbin $(REGISTRY)/kennethreitz/httpbin
	docker push $(REGISTRY)/kennethreitz/httpbin

	docker build -t test-backend:$(IMAGE_TAG) --build-arg ENABLE_PROXY=$(ENABLE_PROXY) ./test/e2e/testbackend
	docker tag test-backend:$(IMAGE_TAG) $(REGISTRY)/test-backend:$(IMAGE_TAG)
	docker push $(REGISTRY)/test-backend:$(IMAGE_TAG)

	docker build -t apache/apisix-ingress-controller:$(IMAGE_TAG) --build-arg ENABLE_PROXY=$(ENABLE_PROXY) .
	docker tag apache/apisix-ingress-controller:$(IMAGE_TAG) $(REGISTRY)/apache/apisix-ingress-controller:$(IMAGE_TAG)
	docker push $(REGISTRY)/apache/apisix-ingress-controller:$(IMAGE_TAG)

	docker pull jmalloc/echo-server:latest
	docker tag  jmalloc/echo-server:latest $(REGISTRY)/jmalloc/echo-server:latest
	docker push $(REGISTRY)/jmalloc/echo-server:latest

	docker pull busybox:1.28
	docker tag  busybox:1.28 $(REGISTRY)/busybox:1.28
	docker push $(REGISTRY)/busybox:1.28
endif

### kind-up:              Launch a Kubernetes cluster with a image registry by Kind.
.PHONY: kind-up
kind-up:
	./utils/kind-with-registry.sh
### kind-reset:           Delete the Kubernetes cluster created by "make kind-up"
.PHONY: kind-reset
kind-reset:
	kind delete cluster --name apisix

### help:                 Show Makefile rules
.PHONY: help
help:
	@echo Makefile rules:
	@echo
	@grep -E '^### [-A-Za-z0-9_]+:' Makefile | sed 's/###/   /'

### release-src:          Release source
.PHONY: release-src
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

.PHONY: gen-tools
gen-tools:
	go mod download
	@bash -c 'go install k8s.io/code-generator/cmd/{client-gen,lister-gen,informer-gen,deepcopy-gen}'

### verify-codegen:       Verify whether the generated codes (clientset, informer, deepcopy, etc) are up to date.
.PHONY: verify-codegen
verify-codegen: gen-tools
	./utils/verify-codegen.sh

### verify-license:       Verify license headers.
.PHONY: verify-license
verify-license:
	docker run -it --rm -v $(PWD):/github/workspace apache/skywalking-eyes header check -v info

### verify-mdlint:        Verify markdown files lint rules.
.PHONY: verify-mdlint
verify-mdlint:
	docker run -it --rm -v $(PWD):/work tmknom/markdownlint '**/*.md' --ignore node_modules

### verify-all:           Verify all verify- rules.
.PHONY: verify-all
verify-all: verify-codegen verify-license verify-mdlint

### update-codegen:       Update the generated codes (clientset, informer, deepcopy, etc).
.PHONY: update-codegen
update-codegen: gen-tools
	./utils/update-codegen.sh

### update-license:       Update license headers.
.PHONY: update-license
update-license:
	docker run -it --rm -v $(PWD):/github/workspace apache/skywalking-eyes header fix

### update-mdlint:        Update markdown files lint rules.
.PHONY: update-mdlint
update-mdlint:
	docker run -it --rm -v $(PWD):/work tmknom/markdownlint '**/*.md' -f --ignore node_modules --ignore vendor

### gofmt:                Format all go codes
.PHONY: update-gofmt
update-gofmt:
	./utils/goimports-reviser.sh

### update-all:           Update all update- rules.
.PHONY: update-all
update-all: update-codegen update-license update-mdlint update-gofmt

### e2e-names-check:          Check if e2e test cases' names have the prefix "suite-<suite-name>".
.PHONY: e2e-names-check
e2e-names-check:
	chmod +x ./utils/check-e2e-names.sh && ./utils/check-e2e-names.sh
