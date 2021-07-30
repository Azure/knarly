# Directories.
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
HACK_DIR := hack
HACK_BIN_DIR := $(abspath $(HACK_DIR)/bin)

KUSTOMIZE_VER := v4.2.0
KUSTOMIZE_BIN := kustomize
KUSTOMIZE := $(HACK_BIN_DIR)/$(KUSTOMIZE_BIN)

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: $(KUSTOMIZE)
$(KUSTOMIZE): ## Build kustomize from tools folder.
	GOBIN=$(HACK_BIN_DIR) go get sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VER)

.PHONY: generate ## Generate template flavors
generate: $(KUSTOMIZE)
	./scripts/gen-flavors.sh