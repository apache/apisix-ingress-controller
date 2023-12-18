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

VERSION ?= 1.7.0


TARGET_APISIX_VERSION ?= "3.4.1-centos"
APISIX_ADMIN_API_VERSION ?= "v3"

ifeq ($(APISIX_ADMIN_API_VERSION),"v2")
    TARGET_APISIX_VERSION ?= "2.15.3-centos"
endif

RELEASE_SRC = apache-apisix-ingress-controller-${VERSION}-src
REPOSITORY="127.0.0.1"
REGISTRY_PORT ?= "5000"
REGISTRY ?="${REPOSITORY}:$(REGISTRY_PORT)"
IMAGE_TAG ?= dev
BASE_IMAGE_TAG ?= nonroot
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
E2E_NODES ?= 4
E2E_FLAKE_ATTEMPTS ?= 0
E2E_SKIP_BUILD ?= 0
# E2E_ENV = "dev"	Keep only failure logs
# E2E_ENV = "ci"	Keep only debug logs
# E2E_ENV = "debug"	Keep only debug logs and testing environment
E2E_ENV ?= "dev"

### build:                Build apisix-ingress-controller
.PHONY: build
build:
	CGO_ENABLED=0 go build \
		-o apisix-ingress-controller \
		-ldflags $(GO_LDFLAGS) \
		main.go

### clean-image:          clean apisix-ingress-controller image
.PHONY: clean-image
clean-image: ## Removes local image
	echo "removing old image $(REGISTRY)/apisix-ingress-controller:$(IMAGE_TAG)"
	docker rmi -f $(REGISTRY)/apisix-ingress-controller:$(IMAGE_TAG) || true

### build-image:          Build apisix-ingress-controller image
.PHONY: build-image
build-image:
ifeq ($(E2E_SKIP_BUILD), 0)
	DOCKER_BUILDKIT=1 docker build -t apache/apisix-ingress-controller:$(IMAGE_TAG) --build-arg ENABLE_PROXY=$(ENABLE_PROXY) --build-arg BASE_IMAGE_TAG=$(BASE_IMAGE_TAG) .
	docker tag apache/apisix-ingress-controller:$(IMAGE_TAG) $(REGISTRY)/apisix-ingress-controller:$(IMAGE_TAG)
endif

### pack-image:   Build and push Ingress image used in e2e test suites to kind or custom registry.
.PHONY: pack-image
pack-image: build-image
	docker push $(REGISTRY)/apisix-ingress-controller:$(IMAGE_TAG)

### pack-images:          Build and push images used in e2e test suites to kind or custom registry.
.PHONY: pack-images
pack-images: build-images push-images

### build-images:         Prepare all required Images
.PHONY: build-images
build-images: build-image
ifeq ($(E2E_SKIP_BUILD), 0)
	docker pull apache/apisix:$(TARGET_APISIX_VERSION)
	docker tag apache/apisix:$(TARGET_APISIX_VERSION) $(REGISTRY)/apisix:$(IMAGE_TAG)

	docker pull bitnami/etcd:3.4.14-debian-10-r0
	docker tag bitnami/etcd:3.4.14-debian-10-r0 $(REGISTRY)/etcd:$(IMAGE_TAG)

	docker pull kennethreitz/httpbin
	docker tag kennethreitz/httpbin $(REGISTRY)/httpbin:$(IMAGE_TAG)

	docker build -t test-backend:$(IMAGE_TAG) --build-arg ENABLE_PROXY=$(ENABLE_PROXY) ./test/e2e/testbackend
	docker build -t test-timeout:$(IMAGE_TAG) --build-arg ENABLE_PROXY=$(ENABLE_PROXY) ./test/e2e/testtimeout	
	docker tag test-backend:$(IMAGE_TAG) $(REGISTRY)/test-backend:$(IMAGE_TAG)
	docker tag test-timeout:$(IMAGE_TAG) $(REGISTRY)/test-timeout:$(IMAGE_TAG)
	docker tag apache/apisix-ingress-controller:$(IMAGE_TAG) $(REGISTRY)/apisix-ingress-controller:$(IMAGE_TAG)

	docker pull jmalloc/echo-server:latest
	docker tag  jmalloc/echo-server:latest $(REGISTRY)/echo-server:$(IMAGE_TAG)

	docker pull busybox:1.28
	docker tag  busybox:1.28 $(REGISTRY)/busybox:$(IMAGE_TAG)
endif

### push-images:          Push images used in e2e test suites to kind or custom registry.
.PHONY: push-images
push-images:
ifeq ($(E2E_SKIP_BUILD), 0)
	docker push $(REGISTRY)/apisix:$(IMAGE_TAG)
	docker push $(REGISTRY)/etcd:$(IMAGE_TAG)
	docker push $(REGISTRY)/httpbin:$(IMAGE_TAG)
	docker push $(REGISTRY)/test-backend:$(IMAGE_TAG)
	docker push $(REGISTRY)/test-timeout:$(IMAGE_TAG)
	docker push $(REGISTRY)/apisix-ingress-controller:$(IMAGE_TAG)
	docker push $(REGISTRY)/echo-server:$(IMAGE_TAG)
	docker push $(REGISTRY)/busybox:$(IMAGE_TAG)
endif

### lint:                 Do static lint check
.PHONY: lint
lint:
	golangci-lint run

### unit-test:            Run unit test cases
.PHONY: unit-test
unit-test:
	go test -cover -coverprofile=coverage.txt ./...

### run-e2e-test          Run e2e test cases only
.PHONY: run-e2e-test
run-e2e-test:
	cd test/e2e \
		&& go mod download \
		&& export REGISTRY=$(REGISTRY) \
		&& APISIX_ADMIN_API_VERSION=$(APISIX_ADMIN_API_VERSION) E2E_ENV=$(E2E_ENV) ACK_GINKGO_RC=true ginkgo -cover -coverprofile=coverage.txt -r --randomize-all --randomize-suites --trace --nodes=$(E2E_NODES) --focus=$(E2E_FOCUS) --flake-attempts=$(E2E_FLAKE_ATTEMPTS)

### e2e-test:             Run e2e test cases (in existing clusters directly)
.PHONY: e2e-test
e2e-test: ginkgo-check pack-images e2e-wolf-rbac e2e-ldap install install-gateway-api run-e2e-test

### e2e-test-local:       Run e2e test cases (kind is required)
.PHONY: e2e-test-local
e2e-test-local: kind-up e2e-test

.PHONY: ginkgo-check
ginkgo-check:
ifeq ("$(wildcard $(GINKGO))", "")
	@echo "ERROR: Need to install ginkgo first, run: go get -u github.com/onsi/ginkgo/v2/ginkgo or go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo"
	exit 1
endif

### install:				Install CRDs into the K8s cluster.
.PHONY: install
install:
	kubectl apply -k $(PWD)/samples/deploy/crd

### uninstall:				Uninstall CRDs from the K8s cluster.
.PHONY: uninstall
uninstall:
	kubectl delete -k $(PWD)/samples/deploy/crd

### kind-up:              Launch a Kubernetes cluster with a image registry by Kind.
.PHONY: kind-up
kind-up:
	REPOSITORY=${REPOSITORY} ./utils/kind-with-registry.sh $(REGISTRY_PORT)

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

	gpg -u zhangjintao@apache.org --batch --yes --armor --detach-sig $(RELEASE_SRC).tgz
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
	docker run -it --rm -v $(PWD):/work tmknom/markdownlint '**/*.md' --ignore node_modules --ignore CHANGELOG.md

### verify-yamllint:	  Verify yaml files lint rules for `samples/deploy` directory.
.PHONY: verify-yamllint
verify-yamllint:
	docker run -it --rm -v $(PWD):/yaml peterdavehello/yamllint yamllint samples/deploy

### verify-all:           Verify all verify- rules.
.PHONY: verify-all
verify-all: verify-codegen verify-license verify-mdlint verify-yamllint

### update-yamlfmt:       Update yaml files format for `samples/deploy` directory.
.PHONY: update-yamlfmt
update-yamlfmt:
	go install github.com/google/yamlfmt/cmd/yamlfmt@latest && yamlfmt samples/deploy

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
	docker run -it --rm -v $(PWD):/work tmknom/markdownlint '**/*.md' -f --ignore node_modules --ignore vendor --ignore CHANGELOG.md

### update-gofmt:         Format all go codes
.PHONY: update-gofmt
update-gofmt:
	./utils/goimports-reviser.sh

### update-all:           Update all update- rules.
.PHONY: update-all
update-all: update-codegen update-license update-mdlint update-gofmt

### e2e-names-check:      Check if e2e test cases' names have the prefix "suite-<suite-name>".
.PHONY: e2e-names-check
e2e-names-check:
	chmod +x ./utils/check-e2e-names.sh && ./utils/check-e2e-names.sh

.PHONY: e2e-wolf-rbac
e2e-wolf-rbac:
ifeq ("$(E2E_FOCUS)", "")
	chmod +x ./test/e2e/testdata/wolf-rbac/cmd.sh && ./test/e2e/testdata/wolf-rbac/cmd.sh start
endif
ifneq ("$(E2E_FOCUS)", "")
	echo $(E2E_FOCUS) | grep -E 'suite-plugins-authentication|consumer|wolf' || exit 0 \
	&& chmod +x ./test/e2e/testdata/wolf-rbac/cmd.sh \
	&& ./test/e2e/testdata/wolf-rbac/cmd.sh start
endif

.PHONY: e2e-ldap
e2e-ldap:
ifeq ("$(E2E_FOCUS)", "")
	chmod +x ./test/e2e/testdata/ldap/cmd.sh && ./test/e2e/testdata/ldap/cmd.sh start
endif
ifneq ("$(E2E_FOCUS)", "")
	echo $(E2E_FOCUS) | grep -E 'suite-plugins-authentication|consumer|ldap' || exit 0 \
	&& chmod +x ./test/e2e/testdata/ldap/cmd.sh \
	&& ./test/e2e/testdata/ldap/cmd.sh start
endif

### kind-load-images:	  Load the images to the kind cluster
.PHONY: kind-load-images
kind-load-images:
	kind load docker-image --name=apisix \
			$(REGISTRY)/apisix:dev \
            $(REGISTRY)/etcd:dev \
            $(REGISTRY)/apisix-ingress-controller:dev \
            $(REGISTRY)/httpbin:dev \
            $(REGISTRY)/test-backend:dev \
			$(REGISTRY)/test-timeout:dev \
            $(REGISTRY)/echo-server:dev \
            $(REGISTRY)/busybox:dev


GATEWAY_API_VERSION ?= v0.6.0
GATEWAY_API_PACKAGE ?= sigs.k8s.io/gateway-api@$(GATEWAY_API_VERSION)
GATEWAY_API_CRDS_GO_MOD_PATH = $(shell go env GOPATH)/pkg/mod/$(GATEWAY_API_PACKAGE)
GATEWAY_API_CRDS_LOCAL_PATH = $(PWD)/samples/deploy/gateway-api/$(GATEWAY_API_VERSION)

.PHONY: go-mod-download-gateway-api
go-mod-download-gateway-api:
	@go mod download $(GATEWAY_API_PACKAGE)

### install-gateway-api:				Install Gateway API into the K8s cluster from go mod.
.PHONY: install-gateway-api
install-gateway-api: go-mod-download-gateway-api
	kubectl apply -k $(GATEWAY_API_CRDS_GO_MOD_PATH)/config/crd
	kubectl apply -k $(GATEWAY_API_CRDS_GO_MOD_PATH)/config/crd/experimental
	kubectl apply -f $(GATEWAY_API_CRDS_GO_MOD_PATH)/config/webhook

### install-gateway-api-local:				Install Gateway API into the K8s cluster from repo.
.PHONY: install-gateway-api-local
install-gateway-api-local:
	kubectl apply -f $(GATEWAY_API_CRDS_LOCAL_PATH)

### uninstall-gateway-api:	Uninstall Gateway API from the K8s cluster.
.PHONY: uninstall-gateway-api
uninstall-gateway-api:
	kubectl delete -f $(GATEWAY_API_CRDS_LOCAL_PATH)
