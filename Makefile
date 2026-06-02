# Tacito Square — Makefile
# Each component has an independent version lifecycle

AGENT_VERSION   := $(shell cat VERSION.agent 2>/dev/null || echo "0.0.1")
KEEPER_VERSION  := $(shell cat VERSION.keeper 2>/dev/null || echo "0.0.1")
OPERATOR_VERSION := $(shell cat VERSION.operator 2>/dev/null || echo "0.0.1")
BFF_VERSION     := $(shell cat VERSION.bff 2>/dev/null || echo "0.0.1")

REGISTRY           ?= 
HELM_RELEASE       ?= ts
HELM_CHART         := tools/helm/tacito-square
HELM_INFRA_RELEASE ?= ts-infra
HELM_INFRA_CHART   := tools/helm/tacito-square-infra
NAMESPACE          ?= tacito
HELM_DEV_VALUES    := tools/helm/dev-values.yaml
HELM_AGENT_RELEASE ?= ts-agent
HELM_AGENT_CHART   := tools/helm/tacito-agent

GO             := go
GOTEST         := $(GO) test
GOLINT         := $(shell which golangci-lint 2>/dev/null || echo "$(shell go env GOPATH 2>/dev/null)/bin/golangci-lint")

NERDCTL_ADDR   := /var/run/docker/containerd/containerd.sock

.PHONY: all build test test-integration test-operator test-e2e test-bench test-race test-contract check-test-tags lint generate \
        escape-analysis escape-agent escape-keeper escape-operator escape-bff \
        docker-build docker-push \
        docker-load docker-load-agent docker-load-keeper docker-load-operator docker-load-bff \
        helm-template helm-install helm-uninstall \
        helm-template-agent helm-install-agent helm-uninstall-agent test-helm-agent \
        helm-infra-deps helm-infra-lint helm-infra-template helm-infra-install helm-infra-uninstall \
        ci clean help

## —— Build ——————————————————————————————————————————————

all: lint test build ## Run lint, test, and build

build: ## Build all binaries
	$(GO) build -o bin/agent    ./cmd/agent
	$(GO) build -o bin/keeper   ./cmd/keeper
	$(GO) build -o bin/operator ./cmd/operator
	$(GO) build -o bin/bff      ./cmd/bff

build-agent: ## Build agent binary
	$(GO) build -o bin/agent ./cmd/agent

build-keeper: ## Build keeper binary
	$(GO) build -o bin/keeper ./cmd/keeper

build-operator: ## Build operator binary
	$(GO) build -o bin/operator ./cmd/operator

build-bff: ## Build bff binary
	$(GO) build -o bin/bff ./cmd/bff

## —— Test ———————————————————————————————————————————————

test: check-test-tags ## Run unit tests with race detector
	$(GOTEST) ./internal/... -count=1 -race -v

check-test-tags: ## Fail if a test file imports testcontainers/dockertest without //go:build integration
	@bad=$$(grep -lE '"github.com/(testcontainers/testcontainers-go|ory/dockertest)' \
		$$(find . -path ./vendor -prune -o -name '*_test.go' -print) 2>/dev/null \
		| xargs -I{} sh -c 'head -n1 "{}" | grep -q "^//go:build integration" || echo "{}"'); \
	if [ -n "$$bad" ]; then \
		echo "ERROR: test files import testcontainers/dockertest without '//go:build integration':"; \
		echo "$$bad" | sed 's/^/  - /'; \
		exit 1; \
	fi

test-integration: ## Run integration tests (requires Docker for testcontainers)
	$(GOTEST) ./... -tags=integration -count=1 -v

test-operator: ## Run operator tests with envtest
	$(GOTEST) ./operator/... -count=1 -v

test-e2e: ## Run end-to-end tests (requires Kind cluster)
	$(GOTEST) ./test/e2e/... -tags=e2e -count=1 -v

test-bench: ## Run benchmark tests
	$(GOTEST) -bench=. -benchmem -run=^$$ ./...

test-race: ## Run all tests with race detector
	$(GOTEST) -race -count=1 ./...

test-contract: ## Run contract tests (OpenAPI compatibility)
	$(GOTEST) ./test/contract/... -tags=integration -count=1 -v

## —— Quality ————————————————————————————————————————————

lint: ## Run linter
	$(GOLINT) run ./...

generate: ## Run code generation (mocks, CRDs, etc.)
	$(GO) generate ./...

## —— Escape Analysis ——————————————————————————————————————

escape-analysis: escape-agent escape-keeper escape-operator escape-bff ## Run escape analysis for all components

escape-agent: ## Run Go escape analysis for the agent component
	@echo "=== Escape Analysis: agent ==="
	$(GO) build -gcflags="-m" ./cmd/agent/... ./internal/agent/... 2>&1 | grep -E "^(\./)?(cmd|internal|pkg)/" || true

escape-keeper: ## Run Go escape analysis for the keeper component
	@echo "=== Escape Analysis: keeper ==="
	$(GO) build -gcflags="-m" ./cmd/keeper/... ./internal/keeper/... 2>&1 | grep -E "^(\./)?(cmd|internal|pkg)/" || true

escape-operator: ## Run Go escape analysis for the operator component
	@echo "=== Escape Analysis: operator ==="
	$(GO) build -gcflags="-m" ./cmd/operator/... ./internal/operator/... 2>&1 | grep -E "^(\./)?(cmd|internal|pkg)/" || true

escape-bff: ## Run Go escape analysis for the bff component
	@echo "=== Escape Analysis: bff ==="
	$(GO) build -gcflags="-m" ./cmd/bff/... ./internal/bff/... 2>&1 | grep -E "^(\./)?(cmd|internal|pkg)/" || true

## —— Docker —————————————————————————————————————————————

docker-build: docker-build-agent docker-build-keeper docker-build-operator docker-build-bff ## Build all Docker images

docker-build-agent: ## Build agent Docker image
	docker build -f tools/docker/Dockerfile.agent -t $(REGISTRY)tacito-square/agent:$(AGENT_VERSION) .

docker-build-keeper: ## Build keeper Docker image
	docker build -f tools/docker/Dockerfile.keeper -t $(REGISTRY)tacito-square/keeper:$(KEEPER_VERSION) .

docker-build-operator: ## Build operator Docker image
	docker build -f tools/docker/Dockerfile.operator -t $(REGISTRY)tacito-square/operator:$(OPERATOR_VERSION) .

docker-build-bff: ## Build bff Docker image
	docker build -f tools/docker/Dockerfile.bff -t $(REGISTRY)tacito-square/bff:$(BFF_VERSION) .

docker-push: ## Push all Docker images
	docker push $(REGISTRY)tacito-square/agent:$(AGENT_VERSION)
	docker push $(REGISTRY)tacito-square/keeper:$(KEEPER_VERSION)
	docker push $(REGISTRY)tacito-square/operator:$(OPERATOR_VERSION)
	docker push $(REGISTRY)tacito-square/bff:$(BFF_VERSION)

docker-load: docker-load-agent docker-load-keeper docker-load-operator docker-load-bff ## Load all images into Rancher Desktop containerd (k8s.io namespace)

docker-load-agent: ## Load agent image into Rancher Desktop containerd
	docker save $(REGISTRY)tacito-square/agent:$(AGENT_VERSION) | rdctl shell -- nerdctl --address $(NERDCTL_ADDR) -n k8s.io load

docker-load-keeper: ## Load keeper image into Rancher Desktop containerd
	docker save $(REGISTRY)tacito-square/keeper:$(KEEPER_VERSION) | rdctl shell -- nerdctl --address $(NERDCTL_ADDR) -n k8s.io load

docker-load-operator: ## Load operator image into Rancher Desktop containerd
	docker save $(REGISTRY)tacito-square/operator:$(OPERATOR_VERSION) | rdctl shell -- nerdctl --address $(NERDCTL_ADDR) -n k8s.io load

docker-load-bff: ## Load bff image into Rancher Desktop containerd
	docker save $(REGISTRY)tacito-square/bff:$(BFF_VERSION) | rdctl shell -- nerdctl --address $(NERDCTL_ADDR) -n k8s.io load

## —— Helm (app) —————————————————————————————————————————

helm-template: ## Render application Helm templates locally
	helm template $(HELM_RELEASE) $(HELM_CHART) --namespace $(NAMESPACE)

helm-install: ## Install/upgrade application Helm release
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) --namespace $(NAMESPACE) --create-namespace --values $(HELM_DEV_VALUES) --wait

helm-uninstall: ## Uninstall application Helm release
	helm uninstall $(HELM_RELEASE) --namespace $(NAMESPACE)

## —— Helm (agent) ———————————————————————————————————————

helm-template-agent: ## Render standalone agent Helm templates locally
	helm template $(HELM_AGENT_RELEASE) $(HELM_AGENT_CHART) --namespace $(NAMESPACE)

helm-install-agent: ## Install/upgrade standalone agent Helm release
	helm upgrade --install $(HELM_AGENT_RELEASE) $(HELM_AGENT_CHART) --namespace $(NAMESPACE) --create-namespace --wait

helm-uninstall-agent: ## Uninstall standalone agent Helm release
	helm uninstall $(HELM_AGENT_RELEASE) --namespace $(NAMESPACE)

test-helm-agent: ## Run automated dry-run testing suite for the standalone agent Helm chart
	bash test/helm/test_agent_standalone_chart.sh

## —— Helm (infra) ———————————————————————————————————————

helm-infra-deps: ## Download infrastructure chart dependencies
	GODEBUG=http2client=0 helm dependency update $(HELM_INFRA_CHART)

helm-infra-lint: ## Lint the infrastructure Helm chart
	helm lint $(HELM_INFRA_CHART)

helm-infra-template: ## Render infrastructure Helm templates locally
	helm template $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) --namespace $(NAMESPACE)

helm-infra-install: ## Install/upgrade infrastructure Helm release
	helm upgrade --install $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) --namespace $(NAMESPACE) --create-namespace --wait

helm-infra-uninstall: ## Uninstall infrastructure Helm release
	helm uninstall $(HELM_INFRA_RELEASE) --namespace $(NAMESPACE)

## —— CI —————————————————————————————————————————————————

ci: lint test test-integration test-contract build docker-build ## Full CI pipeline

## —— Utilities ——————————————————————————————————————————

clean: ## Clean build artifacts
	rm -rf bin/

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
