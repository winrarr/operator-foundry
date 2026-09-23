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
# renovate: datasource=docker depName=kindest/node
KIND_NODE_IMAGE ?= kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5
# renovate: datasource=github-releases depName=cilium/cilium
CILIUM_VERSION ?= 1.20.2
E2E_TEST_NAMESPACE ?= operator-foundry-e2e
SCOPED_KIND_CLUSTER ?= operator-foundry-scoped
SCOPED_NAMESPACE ?= operator-foundry-scope-a
WATCH_NAMESPACES ?= []
# renovate: datasource=docker depName=curlimages/curl
CURL_TEST_IMAGE ?= curlimages/curl:8.22.0@sha256:58adaa4e8dca9c988bae2aba4ab3434a0bb2da16bbe3f92dec39ec7785166777
CONTAINER_TOOL ?= docker
DOCKER_BUILD_CACHE_ARGS ?=
# renovate: datasource=docker depName=zensical/zensical
DOCS_CONTAINER_IMAGE ?= zensical/zensical:0.0.62@sha256:162b7e191224f57b8c584debe51b157b9802efd25d3a8948e4e0f64c1baaaee6
DOCS_CONTAINER_MOUNTS = -v "$(PROJECT_DIR)":/docs
DOCS_CONFIG ?= zensical.toml
# renovate: datasource=docker depName=aquasec/trivy
TRIVY_IMAGE ?= aquasec/trivy:0.74.0@sha256:62b1e65e8869bc4b4c6aa4fa2b21595256c7c2f6018a9d9ad61caf87187c1969
# renovate: datasource=docker depName=anchore/syft
SYFT_IMAGE ?= anchore/syft:v1.51.1@sha256:95fe0835e5bebc6f8b1f8acef68d47d63d594ef4c0f25c097ff853b23cbac74c
TRIVY_CACHE_DIR ?= $(HOME)/.cache/operator-foundry/trivy
SBOM_FILE ?= dist/operator-foundry-image.sbom.spdx.json
SBOM_SOURCE ?= docker:$(IMG)
SYFT_DOCKER_CONFIG_MOUNT =
ifneq ($(wildcard $(HOME)/.docker/config.json),)
SYFT_DOCKER_CONFIG_MOUNT = -v "$(HOME)/.docker:/root/.docker:ro"
endif
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
GOVULNCHECK ?= $(LOCALBIN)/govulncheck
HUBBLE ?= $(LOCALBIN)/hubble

# renovate: datasource=golang-version depName=go
GO_TOOLCHAIN ?= go1.27.1
GO := GOTOOLCHAIN=$(GO_TOOLCHAIN) go
GOFMT := $(shell GOTOOLCHAIN=$(GO_TOOLCHAIN) go env GOROOT)/bin/gofmt
# renovate: datasource=github-releases depName=kubernetes-sigs/kustomize
KUSTOMIZE_VERSION ?= v5.8.1
# renovate: datasource=github-releases depName=kubernetes-sigs/controller-tools
CONTROLLER_TOOLS_VERSION ?= v0.22.0
# renovate: datasource=github-releases depName=elastic/crd-ref-docs
CRD_REF_DOCS_VERSION ?= v0.3.0
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2
# renovate: datasource=github-releases depName=kubernetes-sigs/kind
KIND_VERSION ?= v0.33.0
# renovate: datasource=go depName=golang.org/x/vuln
GOVULNCHECK_VERSION ?= v1.8.0
# renovate: datasource=github-releases depName=cilium/hubble
HUBBLE_VERSION ?= v1.19.4

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

.PHONY: vulnerability-scan
vulnerability-scan: govulncheck ## Scan Go dependencies and repository manifests for high-severity vulnerabilities.
	"$(GOVULNCHECK)" ./...
	@mkdir -p "$(TRIVY_CACHE_DIR)"
	$(CONTAINER_TOOL) run --rm \
		-v "$(PROJECT_DIR):/src:ro" \
		-v "$(TRIVY_CACHE_DIR):/root/.cache" \
		-w /src "$(TRIVY_IMAGE)" fs \
		--scanners vuln,misconfig \
		--severity HIGH,CRITICAL \
		--ignore-unfixed \
		--ignorefile /src/.trivyignore.yaml \
		--exit-code 1 \
		--skip-dirs .git --skip-dirs bin --skip-dirs dist --skip-dirs site .

.PHONY: image-vulnerability-scan
image-vulnerability-scan: ## Scan the image named by IMG for high-severity vulnerabilities.
	@test -n "$(IMG)" || { echo "IMG must not be empty" >&2; exit 1; }
	@mkdir -p "$(TRIVY_CACHE_DIR)"
	$(CONTAINER_TOOL) run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v "$(TRIVY_CACHE_DIR):/root/.cache" \
		"$(TRIVY_IMAGE)" image \
		--scanners vuln \
		--severity HIGH,CRITICAL \
		--ignore-unfixed \
		--exit-code 1 "$(IMG)"

.PHONY: image-sbom
image-sbom: ## Generate an SPDX JSON SBOM for the image named by IMG.
	@test -n "$(IMG)" || { echo "IMG must not be empty" >&2; exit 1; }
	@mkdir -p "$$(dirname "$(SBOM_FILE)")"
	$(CONTAINER_TOOL) run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		$(SYFT_DOCKER_CONFIG_MOUNT) \
		-v "$(PROJECT_DIR):/workspace" \
		"$(SYFT_IMAGE)" "$(SBOM_SOURCE)" \
		--source-name "$(PROJECT_NAME)" \
		-o "spdx-json=/workspace/$(SBOM_FILE)"

.PHONY: security
security: vulnerability-scan docker-build image-vulnerability-scan image-sbom ## Run local source and image security checks.

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
deploy: manifests generate helm-lint helm-upgrade ## Install or upgrade the operator chart from current generated assets.

.PHONY: deploy-committed
deploy-committed: helm-upgrade ## Install or upgrade the operator chart from committed assets.

.PHONY: helm-upgrade
helm-upgrade: ## Install or upgrade the operator chart without generation or validation.
	IMG_REF="$(IMG)"; \
	if [[ "$$IMG_REF" == *@* ]]; then \
		IMG_REPO="$${IMG_REF%@*}"; IMG_DIGEST="$${IMG_REF#*@}"; \
		$(HELM_CMD) upgrade --install "$(PROJECT_NAME)" "$(CHART_DIR)" --namespace "$(OPERATOR_NAMESPACE)" --create-namespace \
			--set-string "image.repository=$$IMG_REPO" --set-string "image.digest=$$IMG_DIGEST" --set-string image.tag="" \
			--set-json 'watchNamespaces=$(WATCH_NAMESPACES)' --wait --timeout 5m; \
	else \
		IMG_LAST="$${IMG_REF##*/}"; \
		if [[ "$$IMG_LAST" == *:* ]]; then IMG_REPO="$${IMG_REF%:*}"; IMG_TAG="$${IMG_REF##*:}"; else IMG_REPO="$$IMG_REF"; IMG_TAG="latest"; fi; \
		$(HELM_CMD) upgrade --install "$(PROJECT_NAME)" "$(CHART_DIR)" --namespace "$(OPERATOR_NAMESPACE)" --create-namespace \
			--set-string "image.repository=$$IMG_REPO" --set-string "image.tag=$$IMG_TAG" \
			--set-json 'watchNamespaces=$(WATCH_NAMESPACES)' --wait --timeout 5m; \
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

.PHONY: install-committed
install-committed: ## Install committed CRD artifacts without regenerating them.
	$(KUBECTL_CMD) apply -f config/crd/bases

.PHONY: kind-deploy-e2e
kind-deploy-e2e: kind-up kind-load-image ## Install the operator from committed artifacts for live E2E checks.
	$(MAKE) KUBECTL_ARGS="--context=kind-$(KIND_CLUSTER)" HELM_ARGS="--kube-context=kind-$(KIND_CLUSTER)" install-committed deploy-committed

.PHONY: kind-e2e
kind-e2e: kind-deploy-e2e ## Spin up Kind, install the operator, and run cluster E2E checks.
	KIND_CLUSTER="$(KIND_CLUSTER)" KIND_CNI="$(KIND_CNI)" PROJECT_NAME="$(PROJECT_NAME)" OPERATOR_NAMESPACE="$(OPERATOR_NAMESPACE)" E2E_TEST_NAMESPACE="$(E2E_TEST_NAMESPACE)" MOCK_API_IMG="$(MOCK_API_IMG)" CURL_TEST_IMAGE="$(CURL_TEST_IMAGE)" KUBECTL="$(KUBECTL)" ./hack/e2e-kind.sh

.PHONY: kind-scoped-e2e
kind-scoped-e2e: ## Exercise namespace-scoped manager RBAC and tenant author permissions in a separate Kind cluster.
	$(MAKE) KIND_CLUSTER="$(SCOPED_KIND_CLUSTER)" kind-up kind-load-image
	$(KUBECTL) --context="kind-$(SCOPED_KIND_CLUSTER)" create namespace "$(SCOPED_NAMESPACE)" --dry-run=client -o yaml | $(KUBECTL) --context="kind-$(SCOPED_KIND_CLUSTER)" apply -f -
	$(MAKE) KIND_CLUSTER="$(SCOPED_KIND_CLUSTER)" KUBECTL_ARGS="--context=kind-$(SCOPED_KIND_CLUSTER)" HELM_ARGS="--kube-context=kind-$(SCOPED_KIND_CLUSTER)" WATCH_NAMESPACES='["$(SCOPED_NAMESPACE)"]' install-committed helm-upgrade
	KIND_CLUSTER="$(SCOPED_KIND_CLUSTER)" PROJECT_NAME="$(PROJECT_NAME)" OPERATOR_NAMESPACE="$(OPERATOR_NAMESPACE)" SCOPED_NAMESPACE="$(SCOPED_NAMESPACE)" KUBECTL="$(KUBECTL)" ./hack/e2e-scoped.sh

.PHONY: kind-refresh
kind-refresh: docker-build kind-load-image kind-restart ## Rebuild, load, and restart the operator in Kind.

.PHONY: kind-restart
kind-restart: ## Restart the operator after loading a mutable local image tag.
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPERATOR_NAMESPACE)" rollout restart deployment -l app.kubernetes.io/instance="$(PROJECT_NAME)"
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n "$(OPERATOR_NAMESPACE)" rollout status deployment -l app.kubernetes.io/instance="$(PROJECT_NAME)" --timeout=5m

.PHONY: kind-hubble-check
kind-hubble-check: hubble ## Verify Cilium and Hubble are available in a Cilium Kind cluster.
	@test "$(KIND_CNI)" = cilium || { echo "Set KIND_CNI=cilium for Hubble checks" >&2; exit 1; }
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n kube-system rollout status daemonset/cilium --timeout=10m
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n kube-system rollout status deployment/cilium-operator --timeout=10m
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n kube-system rollout status deployment/hubble-relay --timeout=10m
	$(KUBECTL) --context="kind-$(KIND_CLUSTER)" -n kube-system get service/hubble-relay

.PHONY: kind-down
kind-down: kind ## Delete only the named disposable Kind cluster.
	"$(KIND)" delete cluster --name "$(KIND_CLUSTER)"

.PHONY: kustomize controller-gen crd-ref-docs golangci-lint govulncheck hubble kind
kustomize: $(KUSTOMIZE)
controller-gen: $(CONTROLLER_GEN)
crd-ref-docs: $(CRD_REF_DOCS)
golangci-lint: $(GOLANGCI_LINT)
govulncheck: $(GOVULNCHECK)
hubble: $(HUBBLE)
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

$(GOVULNCHECK):
	$(call go-install-tool,$@,golang.org/x/vuln/cmd/govulncheck,$(GOVULNCHECK_VERSION))

$(HUBBLE):
	@mkdir -p "$(LOCALBIN)"
	HUBBLE_VERSION="$(HUBBLE_VERSION)" ./hack/install_hubble.sh "$@"

KIND_OS ?= linux
KIND_ARCH ?= $(shell $(GO) env GOARCH)

$(KIND):
	@mkdir -p "$(LOCALBIN)"
	@echo "Downloading kind $(KIND_VERSION)"
	@curl --fail --location --silent --show-error -o "$(KIND).tmp" "https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(KIND_OS)-$(KIND_ARCH)"
	@chmod +x "$(KIND).tmp"
	@mv "$(KIND).tmp" "$(KIND)"
