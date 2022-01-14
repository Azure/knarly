# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL:=/usr/bin/env bash

.DEFAULT_GOAL:=help

GOPATH  := $(shell go env GOPATH)
GOARCH  := $(shell go env GOARCH)
GOOS    := $(shell go env GOOS)
GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Active module mode, as we use go modules to manage dependencies
export GO111MODULE=on

# This option is for running docker manifest command
export DOCKER_CLI_EXPERIMENTAL := enabled

# curl retries
CURL_RETRIES=3

# Directories.
ROOT_DIR :=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
HACK_DIR := hack
HACK_BIN_DIR := $(abspath $(HACK_DIR)/bin)

REGISTRY ?= us.gcr.io/k8s-artifacts-prod/cluster-api-azure
IMAGE_NAME ?= cluster-api-azure-controller
CONTROLLER_IMG ?= $(REGISTRY)/$(IMAGE_NAME)

KUSTOMIZE_VER := v4.2.0
KUSTOMIZE_BIN := kustomize
KUSTOMIZE := $(HACK_BIN_DIR)/$(KUSTOMIZE_BIN)

ENVSUBST_VER := master
ENVSUBST_BIN := envsubst
ENVSUBST := $(HACK_BIN_DIR)/$(ENVSUBST_BIN)

KUBECTL_VER := v1.20.4
KUBECTL_BIN := kubectl
KUBECTL := $(HACK_BIN_DIR)/$(KUBECTL_BIN)-$(KUBECTL_VER)

GINKGO_VER := v1.16.4
GINKGO_BIN := ginkgo
GINKGO := $(HACK_BIN_DIR)/$(GINKGO_BIN)

KIND_VER := v0.11.1
KIND_BIN := kind
KIND := $(HACK_BIN_DIR)/$(KIND_BIN)

# Allow overriding the e2e configurations
GINKGO_NODES ?= 3
GINKGO_NOCOLOR ?= false
GINKGO_ARGS ?=
ARTIFACTS ?= $(ROOT_DIR)/_artifacts
E2E_CONF_FILE ?= $(ROOT_DIR)/test/e2e/config/azure-dev.yaml
E2E_CONF_FILE_ENVSUBST := $(ROOT_DIR)/test/e2e/config/azure-dev-envsubst.yaml
SKIP_CLEANUP ?= false
SKIP_CREATE_MGMT_CLUSTER ?= false
TAG ?= 1.1.0
ARCH ?= amd64

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: $(KUSTOMIZE)
$(KUSTOMIZE): ## Install kustomize
	GOBIN=$(HACK_BIN_DIR) go get sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VER)

.PHONY: $(ENVSUBST)
$(ENVSUBST): ## Install envsubst
	GOBIN=$(HACK_BIN_DIR) go get github.com/drone/envsubst/v2/cmd/envsubst@$(ENVSUBST_VER)

.PHONY: $(GINKGO)
$(GINKGO): ## Install ginkgo
	GOBIN=$(HACK_BIN_DIR) go get github.com/onsi/ginkgo/ginkgo@$(GINKGO_VER)

$(KIND): ## Install KinD
	GOBIN=$(HACK_BIN_DIR) go get sigs.k8s.io/kind@$(KIND_VER)

$(KUBECTL): ## Build kubectl
	mkdir -p $(HACK_BIN_DIR)
	rm -f "$(KUBECTL)*"
	curl --retry $(CURL_RETRIES) -fsL https://storage.googleapis.com/kubernetes-release/release/$(KUBECTL_VER)/bin/$(GOOS)/$(GOARCH)/kubectl -o $(KUBECTL)
	ln -sf "$(KUBECTL)" "$(HACK_BIN_DIR)/$(KUBECTL_BIN)"
	chmod +x "$(HACK_BIN_DIR)/$(KUBECTL_BIN)" "$(KUBECTL)"

.PHONY: generate ## Generate template flavors
generate: $(KUSTOMIZE)
	./scripts/gen-flavors.sh

.PHONY: test-e2e-run
test-e2e-run: generate $(ENVSUBST) $(KUBECTL) $(GINKGO) $(KIND) ## Run e2e tests
	$(ENVSUBST) < $(E2E_CONF_FILE) > $(E2E_CONF_FILE_ENVSUBST) && \
    $(GINKGO) -v -trace -tags=e2e -focus="$(GINKGO_FOCUS)" -skip="$(GINKGO_SKIP)" -nodes=$(GINKGO_NODES) --noColor=$(GINKGO_NOCOLOR) $(GINKGO_ARGS) ./test/e2e -- \
    	-e2e.artifacts-folder="$(ARTIFACTS)" \
    	-e2e.config="$(E2E_CONF_FILE_ENVSUBST)" \
    	-e2e.skip-resource-cleanup=$(SKIP_CLEANUP) -e2e.use-existing-cluster=$(SKIP_CREATE_MGMT_CLUSTER) $(E2E_ARGS)

.PHONY: test-e2e
test-e2e: ## Run e2e tests
	PULL_POLICY=IfNotPresent MANAGER_IMAGE=$(CONTROLLER_IMG):$(TAG) \
	$(MAKE) test-e2e-run
