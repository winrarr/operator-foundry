SHELL := /usr/bin/env bash
.SHELLFLAGS := -o pipefail -ec
.DEFAULT_GOAL := help

PROJECT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
LOCALBIN ?= $(PROJECT_DIR)/bin
PROJECT_NAME ?= operator-foundry
CHART_DIR ?= charts/operator-foundry
IMG ?= ghcr.io/winrarr/operator-foundry:dev
MOCK_API_IMG ?= ghcr.io/winrarr/operator-foundry-mock-api:dev
OPERATOR_NAMESPACE ?= operator-foundry-system
KIND_CLUSTER ?= operator-foundry
KIND_CNI ?= default
KIND_NODE_IMAGE ?= kindest/node:v1.37.0
CILIUM_VERSION ?= 1.20.1
E2E_TEST_NAMESPACE ?= operator-foundry-e2e
CURL_TEST_IMAGE ?= curlimages/curl:8.12.1
CONTAINER_TOOL ?= docker
DOCKER_BUILD_CACHE_ARGS ?=
DOCS_CONTAINER_IMAGE ?= zensical/zensical:0.0.59
DOCS_CONTAINER_MOUNTS = -v "$(PROJECT_DIR)":/docs
DOCS_CONFIG ?= zensical.toml
KUBECTL ?= kubectl
KUBECTL_ARGS ?=
KUBECTL_CMD = $(KUBECTL) $(KUBECTL_ARGS)
HELM ?= helm
HELM_ARGS ?=
HELM_CMD = $(HELM) $(HELM_ARGS)
KUSTOMIZE ?= $(LOCALBIN)/kustomize
KIND ?= $(LOCALBIN)/kind
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

GO_TOOLCHAIN ?= go1.27.1
GO := GOTOOLCHAIN=$(GO_TOOLCHAIN) go
GOFMT := $(shell GOTOOLCHAIN=$(GO_TOOLCHAIN) go env GOROOT)/bin/gofmt
KUSTOMIZE_VERSION ?= v5.8.1
CONTROLLER_TOOLS_VERSION ?= v0.22.0
CRD_REF_DOCS_VERSION ?= v0.3.0
GOLANGCI_LINT_VERSION ?= v2.13.2
KIND_VERSION ?= v0.33.0

.PHONY: all
all: check build ## Run verification and build the manager.

##@ Help

.PHONY: help
help: ## Display available targets and variables.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: format
format: ## Format Go sources.
	$(GO) fmt ./...

.PHONY: format-check
format-check: ## Fail when Go sources are not gofmt-clean.
	@files="$$($(GOFMT) -l $$(find api cmd internal -name '*.go' -type f))"; \
	if [ -n "$$files" ]; then echo "Go sources need formatting:" >&2; echo "$$files" >&2; exit 1; fi

.PHONY: generate
generate: controller-gen crd-ref-docs ## Generate deepcopy code and the API reference.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt",year=$(shell date +%Y) paths="./..."
	$(MAKE) generate-api-reference

.PHONY: manifests
manifests: controller-gen ## Generate CRDs and RBAC from API and controller markers.
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd paths="./..." output:crd:artifacts:config=config/crd/bases output:rbac:artifacts:config=config/rbac
	$(MAKE) sync-chart-generated

.PHONY: sync-chart-generated
sync-chart-generated: ## Sync generated CRDs and manager RBAC into the chart.
	./hack/sync-chart-generated.sh

.PHONY: verify-generated
verify-generated: manifests generate ## Verify committed generated artifacts are current.
	@if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then \
		git diff --exit-code -- api/patterns/v1alpha1/zz_generated.deepcopy.go config/crd/bases config/rbac/role.yaml charts/$(PROJECT_NAME)/crds charts/$(PROJECT_NAME)/templates/clusterrole.yaml docs/reference/api.md; \
	else \
		echo "No Git checkout detected; generated files were regenerated but cannot be compared"; \
	fi

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet ./...

.PHONY: test
test: format-check vet ## Run unit and contract tests.
	$(GO) test ./... -coverprofile=cover.out

.PHONY: lint-config
lint-config: golangci-lint ## Validate the linter configuration.
	"$(GOLANGCI_LINT)" config verify

.PHONY: lint
lint: golangci-lint ## Run golangci-lint.
	"$(GOLANGCI_LINT)" run

.PHONY: helm-lint
helm-lint: ## Lint the operator chart.
	$(HELM) lint $(CHART_DIR)

.PHONY: helm-template
helm-template: ## Render the operator chart.
	$(HELM) template $(PROJECT_NAME) $(CHART_DIR) --namespace $(OPERATOR_NAMESPACE) --include-crds >/dev/null

.PHONY: kustomize-build
kustomize-build: kustomize ## Render the default installation bundle.
	"$(KUSTOMIZE)" build config/default >/dev/null

.PHONY: generate-api-reference
generate-api-reference: crd-ref-docs ## Generate the CRD API reference.
	@mkdir -p docs/reference
	"$(CRD_REF_DOCS)" --config hack/crd-ref-docs.yaml --renderer markdown --source-path ./api --output-path docs/reference/api.md
	@awk '{ lines[NR] = $$0 } END { last = NR; while (last > 0 && lines[last] == "") last--; for (i = 1; i <= last; i++) print lines[i] }' docs/reference/api.md > docs/reference/api.md.tmp
	@mv docs/reference/api.md.tmp docs/reference/api.md

.PHONY: build-docs-site
build-docs-site: generate-api-reference ## Build the strict documentation site.
	$(CONTAINER_TOOL) run --rm --workdir /docs $(DOCS_CONTAINER_MOUNTS) $(DOCS_CONTAINER_IMAGE) build --strict --config-file $(DOCS_CONFIG)

.PHONY: docs-build
docs-build: build-docs-site ## Generate the API reference and build the docs site.

.PHONY: docs-serve
docs-serve: generate-api-reference ## Serve the documentation site locally.
	$(CONTAINER_TOOL) run --rm --workdir /docs -p 8000:8000 $(DOCS_CONTAINER_MOUNTS) $(DOCS_CONTAINER_IMAGE) serve --dev-addr 0.0.0.0:8000 --config-file $(DOCS_CONFIG)

.PHONY: shell-check
shell-check: ## Validate repository shell scripts parse successfully.
	bash -n hack/*.sh

.PHONY: check
check: manifests generate format-check shell-check vet test lint-config lint helm-lint helm-template kustomize-build docs-build ## Run the complete local verification suite.

##@ Build

.PHONY: build
build: manifests generate format-check vet ## Build the controller binary.
	@mkdir -p bin
	$(GO) build -trimpath -ldflags="-s -w" -o bin/manager ./cmd

.PHONY: run
run: manifests generate ## Run the controller against the current kubeconfig context.
	$(GO) run ./cmd

.PHONY: docker-build
docker-build: ## Build the operator image.
	$(CONTAINER_TOOL) buildx build --load $(DOCKER_BUILD_CACHE_ARGS) --provenance=false --sbom=false --tag $(IMG) .

.PHONY: docker-build-mock-api
docker-build-mock-api: ## Build the disposable external API fixture image.
	$(CONTAINER_TOOL) buildx build --load $(DOCKER_BUILD_CACHE_ARGS) --provenance=false --sbom=false --target mock-api-runtime --tag $(MOCK_API_IMG) .

.PHONY: docker-buildx
docker-buildx: ## Build and push a multi-platform operator image.
	$(CONTAINER_TOOL) buildx build --platform=$(PLATFORMS) --tag $(IMG) --push .

PLATFORMS ?= linux/amd64,linux/arm64

.PHONY: docker-push
docker-push: ## Push the operator image.
	$(CONTAINER_TOOL) push $(IMG)

.PHONY: build-installer
build-installer: manifests generate kustomize ## Build a standalone Kustomize installation bundle.
	@mkdir -p dist
	"$(KUSTOMIZE)" build config/default | sed 's#image: controller:latest#image: $(IMG)#' > dist/install.yaml

.PHONY: helm-package
helm-package: manifests helm-lint ## Package the operator chart.
	@mkdir -p dist
	"$(HELM)" package "$(CHART_DIR)" --destination dist

##@ Kubernetes

.PHONY: install
install: manifests kustomize ## Install CRDs into the current Kubernetes context.
	"$(KUSTOMIZE)" build config/crd | $(KUBECTL_CMD) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Remove CRDs from the current Kubernetes context.
	"$(KUSTOMIZE)" build config/crd | $(KUBECTL_CMD) delete --ignore-not-found=true -f -

.PHONY: deploy
deploy: manifests generate helm-lint ## Install or upgrade the operator chart.
	IMG_REF="$(IMG)"; \
	if [[ "$$IMG_REF" == *@* ]]; then \
		IMG_REPO="$${IMG_REF%@*}"; IMG_DIGEST="$${IMG_REF#*@}"; \
		$(HELM_CMD) upgrade --install "$(PROJECT_NAME)" "$(CHART_DIR)" --namespace "$(OPERATOR_NAMESPACE)" --create-namespace \
			--set-string "image.repository=$$IMG_REPO" --set-string "image.digest=$$IMG_DIGEST" --set-string image.tag="" --wait --timeout 5m; \
	else \
		IMG_LAST="$${IMG_REF##*/}"; \
		if [[ "$$IMG_LAST" == *:* ]]; then IMG_REPO="$${IMG_REF%:*}"; IMG_TAG="$${IMG_REF##*:}"; else IMG_REPO="$$IMG_REF"; IMG_TAG="latest"; fi; \
		$(HELM_CMD) upgrade --install "$(PROJECT_NAME)" "$(CHART_DIR)" --namespace "$(OPERATOR_NAMESPACE)" --create-namespace \
			--set-string "image.repository=$$IMG_REPO" --set-string "image.tag=$$IMG_TAG" --wait --timeout 5m; \
	fi

.PHONY: undeploy
undeploy: ## Uninstall the operator chart.
	$(HELM_CMD) uninstall "$(PROJECT_NAME)" --namespace "$(OPERATOR_NAMESPACE)" --ignore-not-found

##@ Local Kind environment

.PHONY: kind-up
kind-up: kind-install-cni ## Create the disposable Kind cluster with the selected CNI.

.PHONY: kind-create
kind-create: kind ## Create the isolated Kind cluster if it does not exist.
	@case "$(KIND_CNI)" in \
		default) kind_config=hack/kind-configuration.yaml ;; \
		cilium) kind_config=hack/kind-configuration-cilium.yaml ;; \
		*) echo "KIND_CNI must be 'default' or 'cilium'" >&2; exit 1 ;; \
	esac; \
	if ! "$(KIND)" get clusters | grep -Fxq "$(KIND_CLUSTER)"; then \
		"$(KIND)" create cluster --name "$(KIND_CLUSTER)" --image "$(KIND_NODE_IMAGE)" --config "$$kind_config"; \
	else \
		echo "Kind cluster $(KIND_CLUSTER) already exists"; \
	fi

.PHONY: kind-ensure-cni
kind-ensure-cni: kind-create ## Verify that the existing Kind cluster matches KIND_CNI.
	@case "$(KIND_CNI)" in \
		default) \
			if "$(KUBECTL)" --context="kind-$(KIND_CLUSTER)" -n kube-system get daemonset kindnet >/dev/null 2>&1; then echo "Using Kind's default CNI"; else echo "Cluster is not using the default CNI; run make kind-down before switching" >&2; exit 1; fi ;; \
		cilium) \
			if "$(KUBECTL)" --context="kind-$(KIND_CLUSTER)" -n kube-system get daemonset kindnet >/dev/null 2>&1; then echo "Cluster still has Kind's default CNI; run make kind-down before switching to Cilium" >&2; exit 1; else echo "Cluster is configured for Cilium"; fi ;; \
		*) echo "KIND_CNI must be 'default' or 'cilium'" >&2; exit 1 ;; \
	esac

.PHONY: kind-install-cni
kind-install-cni: kind-ensure-cni ## Install the selected CNI when required.
	@case "$(KIND_CNI)" in \
		default) ;; \
		cilium) $(MAKE) kind-install-cilium ;; \
		*) echo "KIND_CNI must be 'default' or 'cilium'" >&2; exit 1 ;; \
	esac

.PHONY: kind-install-cilium
kind-install-cilium: ## Install Cilium with Hubble into the Kind cluster.
	"$(HELM)" upgrade --install cilium oci://quay.io/cilium/charts/cilium \
		--version "$(CILIUM_VERSION)" --namespace kube-system --kube-context "kind-$(KIND_CLUSTER)" \
		--set kubeProxyReplacement=true --set k8sServiceHost="$(KIND_CLUSTER)-control-plane" --set k8sServicePort=6443 \
		--set operator.replicas=1 --set hubble.enabled=true --set hubble.relay.enabled=true --set hubble.ui.enabled=false \
		--wait --timeout=10m

.PHONY: kind-load-image
kind-load-image: kind-create docker-build docker-build-mock-api ## Build and load the operator and mock API images into Kind.
	"$(KIND)" load docker-image "$(IMG)" --name "$(KIND_CLUSTER)"
	"$(KIND)" load docker-image "$(MOCK_API_IMG)" --name "$(KIND_CLUSTER)"

.PHONY: kind-deploy
kind-deploy: kind-up kind-load-image ## Install the operator chart into Kind.
	$(MAKE) KUBECTL_ARGS="--context=kind-$(KIND_CLUSTER)" HELM_ARGS="--kube-context=kind-$(KIND_CLUSTER)" install deploy

.PHONY: kind-e2e
kind-e2e: kind-deploy ## Spin up Kind, install the operator, and run cluster E2E checks.
	KIND_CLUSTER="$(KIND_CLUSTER)" KIND_CNI="$(KIND_CNI)" PROJECT_NAME="$(PROJECT_NAME)" OPERATOR_NAMESPACE="$(OPERATOR_NAMESPACE)" E2E_TEST_NAMESPACE="$(E2E_TEST_NAMESPACE)" MOCK_API_IMG="$(MOCK_API_IMG)" CURL_TEST_IMAGE="$(CURL_TEST_IMAGE)" KUBECTL="$(KUBECTL)" ./hack/e2e-kind.sh

.PHONY: kind-refresh
kind-refresh: docker-build kind-load-image ## Rebuild and restart the operator in Kind.
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPERATOR_NAMESPACE)" rollout restart deployment -l app.kubernetes.io/instance="$(PROJECT_NAME)"
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPERATOR_NAMESPACE)" rollout status deployment -l app.kubernetes.io/instance="$(PROJECT_NAME)" --timeout=5m

.PHONY: kind-hubble-check
kind-hubble-check: ## Verify Cilium and Hubble are available in a Cilium Kind cluster.
	@test "$(KIND_CNI)" = cilium || { echo "Set KIND_CNI=cilium for Hubble checks" >&2; exit 1; }
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n kube-system rollout status daemonset/cilium --timeout=10m
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n kube-system rollout status deployment/cilium-operator --timeout=10m
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n kube-system rollout status deployment/hubble-relay --timeout=10m
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n kube-system get service/hubble-relay

.PHONY: kind-down
kind-down: kind ## Delete only the named disposable Kind cluster.
	"$(KIND)" delete cluster --name "$(KIND_CLUSTER)"

.PHONY: kustomize controller-gen crd-ref-docs golangci-lint kind
kustomize: $(KUSTOMIZE)
controller-gen: $(CONTROLLER_GEN)
crd-ref-docs: $(CRD_REF_DOCS)
golangci-lint: $(GOLANGCI_LINT)
kind: $(KIND)

define go-install-tool
	@mkdir -p "$(LOCALBIN)"
	GOBIN="$(LOCALBIN)" $(GO) install $(2)@$(3)
endef

$(KUSTOMIZE):
	$(call go-install-tool,$@,sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

$(CONTROLLER_GEN):
	$(call go-install-tool,$@,sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

$(CRD_REF_DOCS):
	$(call go-install-tool,$@,github.com/elastic/crd-ref-docs,$(CRD_REF_DOCS_VERSION))

$(GOLANGCI_LINT):
	$(call go-install-tool,$@,github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

KIND_OS ?= linux
KIND_ARCH ?= $(shell $(GO) env GOARCH)

$(KIND):
	@mkdir -p "$(LOCALBIN)"
	@echo "Downloading kind $(KIND_VERSION)"
	@curl --fail --location --silent --show-error -o "$(KIND).tmp" "https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(KIND_OS)-$(KIND_ARCH)"
	@chmod +x "$(KIND).tmp"
	@mv "$(KIND).tmp" "$(KIND)"
