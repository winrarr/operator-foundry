# Local configuration for the root Makefile.
#
# Keep supported command overrides and pinned fixture/tool defaults here so
# the Makefile remains focused on workflow orchestration. Override ?= values
# on the command line when a local environment needs different settings.

PROJECT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)
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
