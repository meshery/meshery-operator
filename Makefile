# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
# Current Operator version
VERSION ?= 0.0.1
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
BIN_DIR := $(PROJECT_DIR)/bin

# Keep ENVTEST_K8S_VERSION aligned with the k8s.io/* libraries in go.mod (v0.35.x).
ENVTEST_K8S_VERSION = 1.35.0
# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# meshery/meshery-operator-bundle:$VERSION and meshery/meshery-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= meshery/meshery-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
IMG ?= meshery/meshery-operator:stable-latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
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
	$(CONTROLLER_GEN) rbac:roleName=operator-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: crds
crds: manifests kustomize ## Render distributable CRD bundles into dist/ (plain + webhook-conversion variants).
	mkdir -p dist
	{ echo '# Code generated by "make crds" from config/crd/bases. DO NOT EDIT.'; \
	  echo '# Conversion strategy: None (apiextensions default). Correct while the v1alpha1 and'; \
	  echo '# v1alpha2 schemas are field-identical; see docs/development.md.'; \
	  for f in config/crd/bases/*.yaml; do echo "---"; cat $$f; done; } > dist/crds.yaml
	{ echo '# Code generated by "make crds" (kustomize build config/crd). DO NOT EDIT.'; \
	  echo '# Conversion is wired to the webhook Service meshery-webhook-service in namespace'; \
	  echo '# meshery with cert-manager CA injection; requires the operator webhook and cert-manager.'; \
	  $(KUSTOMIZE) build config/crd; } > dist/crds-webhook-conversion.yaml

# Pinned NATS Helm chart version. Bump deliberately; `make nats-manifests` then
# regenerates the vendored manifests and the drift gate keeps them in sync.
NATS_CHART_VERSION ?= 2.14.2

.PHONY: nats-manifests
nats-manifests: ## Re-render the vendored NATS server manifests from the official chart. Requires the helm CLI (a build/dev tool only — NOT a runtime/go.mod dependency); the operator embeds and SSA-applies the rendered output.
	helm repo add nats https://nats-io.github.io/k8s/helm/charts/ >/dev/null 2>&1 || true
	helm repo update nats >/dev/null
	@printf '%s\n' \
		'# Code generated by "make nats-manifests" (helm template nats/nats --version $(NATS_CHART_VERSION)). DO NOT EDIT.' \
		'# To change: edit pkg/broker/chart/values.yaml (or bump NATS_CHART_VERSION) and re-run make nats-manifests.' \
		> pkg/broker/manifests/nats.gen.yaml
	helm template meshery-nats nats/nats --version $(NATS_CHART_VERSION) \
		--namespace meshery --skip-tests -f pkg/broker/chart/values.yaml \
		>> pkg/broker/manifests/nats.gen.yaml

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

# Run go lint against code
.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run -c .golangci.yml ./...

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run -c .golangci.yml --fix ./...

.PHONY: tidy
tidy: ## Run go mod tidy against code.
	go mod tidy

.PHONY: error
error: ## Analyze MeshKit error codes (read-only); writes errorutil_*.json to ./helpers.
	go run github.com/meshery/meshkit/cmd/errorutil -d . analyze -i ./helpers -o ./helpers

.PHONY: error-util
error-util: ## Assign/update MeshKit error codes and regenerate the error reference export.
	go run github.com/meshery/meshkit/cmd/errorutil -d . update -i ./helpers -o ./helpers

.PHONY: test
test: manifests generate fmt vet test-env ## Run tests.
	KUBEBUILDER_ASSETS="$$(bin/setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir $(BIN_DIR) -p path)" \
		go test --short ./... -race -coverprofile=coverage.txt -covermode=atomic

##@ Build

.PHONY: build
build: generate fmt vet manifests ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	docker buildx create --name project-v3-builder --use
	docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile .
	docker buildx rm project-v3-builder

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image meshery/meshery-operator=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.8.1
CONTROLLER_TOOLS_VERSION ?= v0.18.0
GOLANGCI_LINT_VERSION ?= v2.12.2

# go-install-versioned installs a Go-based tool at a pinned version into bin/,
# re-installing when the on-disk binary reports a different version. The plain
# binary path (no version suffix) is kept so consumers like
# integration-tests/main.sh can reference $(LOCALBIN)/<tool> directly.
# $(1)=binary path  $(2)=version string  $(3)=`go install` package@version  $(4)=version-probe command
define go-install-versioned
@{ \
set -e ;\
if [ ! -x "$(1)" ] || ! $(4) 2>/dev/null | grep -qF "$(patsubst v%,%,$(2))"; then \
	echo "Installing $(3)" ;\
	rm -f "$(1)" ;\
	for attempt in 1 2 3; do \
		if GOBIN=$(LOCALBIN) go install $(3); then break; fi ;\
		echo "go install $(3) failed (attempt $$attempt/3); retrying..." ;\
		sleep $$((attempt * 10)) ;\
	done ;\
	test -x "$(1)" ;\
fi ;\
}
endef

.PHONY: kustomize
kustomize: $(LOCALBIN) ## Download kustomize locally if necessary.
	$(call go-install-versioned,$(KUSTOMIZE),$(KUSTOMIZE_VERSION),sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION),$(KUSTOMIZE) version)

.PHONY: controller-gen
controller-gen: $(LOCALBIN) ## Download controller-gen locally if necessary.
	$(call go-install-versioned,$(CONTROLLER_GEN),$(CONTROLLER_TOOLS_VERSION),sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION),$(CONTROLLER_GEN) --version)

.PHONY: golangci-lint
golangci-lint: $(LOCALBIN) ## Download golangci-lint locally if necessary.
	$(call go-install-versioned,$(GOLANGCI_LINT),$(GOLANGCI_LINT_VERSION),github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION),$(GOLANGCI_LINT) version)

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image meshery/meshery-operator=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle $(BUNDLE_GEN_FLAGS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
OPM_VERSION ?= v1.72.0
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/$(OPM_VERSION)/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

# Test coverage
.PHONY: coverage
coverage: test-env
	go test -v ./... -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

KIND ?= $(LOCALBIN)/kind
KIND_VERSION ?= v0.32.0
.PHONY: kind
kind: $(LOCALBIN) ## Download kind locally (bin/) for the current OS/ARCH if necessary.
	@test -x $(KIND) && $(KIND) version 2>/dev/null | grep -qF "$(patsubst v%,%,$(KIND_VERSION))" || { \
		echo "Installing kind $(KIND_VERSION)" ;\
		curl -fsSLo $(KIND) https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(shell go env GOOS)-$(shell go env GOARCH) ;\
		chmod +x $(KIND) ;\
	}

SETUP_ENVTEST_VERSION := v0.24.1

bin/setup-envtest: $(BIN_DIR)/setup-envtest-$(SETUP_ENVTEST_VERSION) ## Install setup-envtest CLI
	@ln -sf setup-envtest-$(SETUP_ENVTEST_VERSION) $(BIN_DIR)/setup-envtest

$(BIN_DIR)/setup-envtest-$(SETUP_ENVTEST_VERSION):
	@mkdir -p $(BIN_DIR)
	@GOBIN=$(BIN_DIR) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)
	@mv $(BIN_DIR)/setup-envtest $(BIN_DIR)/setup-envtest-$(SETUP_ENVTEST_VERSION)

# Setting test envrioment
.PHONY: test-env
test-env:
	make bin/setup-envtest
	bin/setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir $(BIN_DIR)

##@ Integration Tests

.PHONY: integration-tests-check-dependencies
## Runs integration tests check dependencies (if docker, kind, kubectl are present)
integration-tests-check-dependencies:
	./integration-tests/main.sh check_dependencies

.PHONY: integration-tests-setup
## Runs integration tests set up (creates kind cluster, builds and deploys operator)
integration-tests-setup:
	./integration-tests/main.sh setup

.PHONY: integration-tests-cleanup
## Runs integration tests clean up (stops cluster and removes operator image)
integration-tests-cleanup:
	./integration-tests/main.sh cleanup

.PHONY: integration-tests-run
## Runs integration tests (validates that meshsync and broker are deployed properly)
integration-tests-run:
	./integration-tests/main.sh assert

.PHONY: integration-tests-setup-debug-output
## Debug integration tests by outputting cluster state
integration-tests-setup-debug-output:
	./integration-tests/main.sh debug

.PHONY: integration-tests
## Runs integration tests full cycle (setup, run validation, cleanup)
integration-tests: integration-tests-setup integration-tests-run integration-tests-cleanup

.PHONY: e2e-dev
## Fast local e2e loop: reuse the kind cluster, rebuild/reload the operator, re-assert
e2e-dev:
	REUSE_CLUSTER=1 ./integration-tests/main.sh setup
	./integration-tests/main.sh assert
