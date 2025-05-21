# Image URL to use all building/pushing image targets

VERSION ?= 2.0.0


IMAGE_TAG ?= dev

IMG ?= apache/apisix-ingress-controller:$(IMAGE_TAG)
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.30.0

KIND_NAME ?= apisix-ingress-cluster
GATEAY_API_VERSION ?= v1.2.0

TEST_TIMEOUT ?= 45m

# CRD Reference Documentation
CRD_REF_DOCS_VERSION ?= v0.1.0
CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs
CRD_DOCS_CONFIG ?= docs/crd/config.yaml
CRD_DOCS_OUTPUT ?= docs/crd/api.md

export KUBECONFIG = /tmp/$(KIND_NAME).kubeconfig

# go 
VERSYM="github.com/apache/apisix-ingress-controller/internal/version._buildVersion"
GITSHASYM="github.com/apache/apisix-ingress-controller/internal/version._buildGitRevision"
BUILDOSSYM="github.com/apache/apisix-ingress-controller/internal/version._buildOS"
GO_LDFLAGS ?= "-X=$(VERSYM)=$(VERSION) -X=$(GITSHASYM)=$(GITSHA) -X=$(BUILDOSSYM)=$(OSNAME)/$(OSARCH)"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
GOOS ?= linux
GOARCH ?= amd64

ifeq ($(shell uname -s),Darwin)
	GOOS = darwin
endif

ifeq ($(shell uname -m),arm64)
	GOARCH = arm64
endif
ifeq ($(shell uname -m), aarch64)
	GOARCH = arm64
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=apisix-ingress-manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e | grep -v /conformance) -coverprofile cover.out

.PHONY: kind-e2e-test
kind-e2e-test: kind-up build-image kind-load-images e2e-test

.PHONY: lint
lint: sort-import golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: kind-up
kind-up:
	@kind get clusters 2>&1 | grep -v $(KIND_NAME) \
		&& kind create cluster --name $(KIND_NAME) \
		|| echo "kind cluster already exists"
	@kind get kubeconfig --name $(KIND_NAME) > $$KUBECONFIG
	kubectl wait --for=condition=Ready nodes --all

.PHONY: kind-down
kind-down:
	@kind get clusters 2>&1 | grep $(KIND_NAME) \
		&& kind delete cluster --name $(KIND_NAME) \
		|| echo "kind cluster does not exist"

.PHONY: kind-load-images
kind-load-images: pull-infra-images kind-load-ingress-image
	@kind load docker-image kennethreitz/httpbin:latest --name $(KIND_NAME) 
	@kind load docker-image jmalloc/echo-server:latest --name $(KIND_NAME)

.PHONY: kind-load-ingress-image
kind-load-ingress-image:
	@kind load docker-image $(IMG) --name $(KIND_NAME)

.PHONY: pull-infra-images
pull-infra-images:
	@docker pull kennethreitz/httpbin:latest
	@docker pull jmalloc/echo-server:latest

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o bin/apisix-ingress-controller_$(GOARCH) -ldflags $(GO_LDFLAGS) cmd/main.go

linux-build:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/apisix-ingress-controller -ldflags $(GO_LDFLAGS) cmd/main.go

.PHONY: build-image
build-image: docker-build

.PHONY: build-push-image
build-push-image: docker-build
	@docker push ${IMG}

.PHONY: build-multi-arch
build-multi-arch:
	@CGO_ENABLED=0 GOARCH=amd64 go build -o bin/apisix-ingress-controller_amd64 -ldflags $(GO_LDFLAGS) cmd/main.go
	@CGO_ENABLED=0 GOARCH=arm64 go build -o bin/apisix-ingress-controller_arm64 -ldflags $(GO_LDFLAGS) cmd/main.go

.PHONY: build-multi-arch-image
build-multi-arch-image: build-multi-arch
    # daemon.json: "features":{"containerd-snapshotter": true}
	@docker buildx build --load --platform linux/amd64,linux/arm64 -t $(IMG) .

.PHONY: build-push-multi-arch-image
build-push-multi-arch-image: build-multi-arch
	@docker buildx build --push --platform linux/amd64,linux/arm64 -t $(IMG) .

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: set-e2e-goos build ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} -f Dockerfile .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name apisix-ingress-builder
	$(CONTAINER_TOOL) buildx use apisix-ingress-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm apisix-ingress-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install-gateway-api
install-gateway-api: ## Install Gateway API CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEAY_API_VERSION)/standard-install.yaml

.PHONY: uninstall-gateway-api
uninstall-gateway-api: ## Uninstall Gateway API CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	kubectl delete -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEAY_API_VERSION)/standard-install.yaml

.PHONY: install
install: manifests kustomize install-gateway-api ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.4.2
CONTROLLER_TOOLS_VERSION ?= v0.15.0
ENVTEST_VERSION ?= release-0.18
GOLANGCI_LINT_VERSION ?= v2.1.5

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s $(GOLANGCI_LINT_VERSION)

gofmt: ## Apply go fmt
	@gofmt -w -r 'interface{} -> any' .
	@go fmt ./...
.PHONY: gofmt

set-e2e-goos:
	$(eval GOOS=linux)
	@echo "e2e GOOS: $(GOOS)"
.PHONY: set-e2e-goos

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

helm-build-crds:
	@echo "build gateway-api standard crds"
	$(KUSTOMIZE) build github.com/kubernetes-sigs/gateway-api/config/crd\?ref=${GATEAY_API_VERSION} > charts/crds/gwapi-crds.yaml
	@echo "build apisix ic crds"
	$(KUSTOMIZE) build config/crd > charts/crds/apisixic-crds.yaml

sort-import:
	@./scripts/goimports-reviser.sh >/dev/null 2>&1
.PHONY: sort-import

# check copyright header
check-copyright:
	go run scripts/go-copyright/main.go

.PHONY: generate-crd-docs
generate-crd-docs: manifests ## Generate CRD reference documentation in a single file
	@mkdir -p $(dir $(CRD_DOCS_OUTPUT))
	@echo "Generating CRD reference documentation"
	@$(CRD_REF_DOCS) \
		--source-path=./api \
		--config=$(CRD_DOCS_CONFIG) \
		--renderer=markdown \
		--templates-dir=./docs/template \
		--output-path=$(CRD_DOCS_OUTPUT) \
		--max-depth=100
	@echo "CRD reference documentation generated at $(CRD_DOCS_OUTPUT)"

.PHONY: generate-crd-docs-grouped
generate-crd-docs-grouped: manifests ## Generate CRD reference documentation grouped by API group
	@mkdir -p docs/crd/groups
	@echo "Generating CRD reference documentation (grouped by API)"
	@$(CRD_REF_DOCS) \
		--source-path=./api \
		--config=$(CRD_DOCS_CONFIG) \
		--renderer=markdown \
		--templates-dir=./docs/template \
		--output-path=docs/crd/groups \
		--output-mode=group
	@echo "CRD reference documentation generated in docs/crd/groups directory"
