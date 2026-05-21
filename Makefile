# Tacito Square — Makefile
# Each component has an independent version lifecycle

AGENT_VERSION   := $(shell cat VERSION.agent 2>/dev/null || echo "0.0.1")
KEEPER_VERSION  := $(shell cat VERSION.keeper 2>/dev/null || echo "0.0.1")
OPERATOR_VERSION := $(shell cat VERSION.operator 2>/dev/null || echo "0.0.1")
BFF_VERSION     := $(shell cat VERSION.bff 2>/dev/null || echo "0.0.1")

REGISTRY       ?= localhost:5000/tacito-square
HELM_RELEASE       ?= tacito-square
HELM_CHART         := tools/helm/tacito-square
HELM_INFRA_RELEASE ?= tacito-infra
HELM_INFRA_CHART   := tools/helm/tacito-square-infra

GO             := go
GOTEST         := $(GO) test
GOLINT         := golangci-lint

.PHONY: all build test test-integration test-operator test-e2e test-bench test-race test-contract lint generate \
        docker-build docker-push \
        helm-template helm-install helm-uninstall \
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

test: ## Run unit tests with race detector
	$(GOTEST) ./internal/... -count=1 -race -v

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
	$(GOTEST) ./test/contract/... -count=1 -v

## —— Quality ————————————————————————————————————————————

lint: ## Run linter
	$(GOLINT) run ./...

generate: ## Run code generation (mocks, CRDs, etc.)
	$(GO) generate ./...

## —— Docker —————————————————————————————————————————————

docker-build: docker-build-agent docker-build-keeper docker-build-operator docker-build-bff ## Build all Docker images

docker-build-agent: ## Build agent Docker image
	docker build -f tools/docker/Dockerfile.agent -t $(REGISTRY)/agent:$(AGENT_VERSION) .

docker-build-keeper: ## Build keeper Docker image
	docker build -f tools/docker/Dockerfile.keeper -t $(REGISTRY)/keeper:$(KEEPER_VERSION) .

docker-build-operator: ## Build operator Docker image
	docker build -f tools/docker/Dockerfile.operator -t $(REGISTRY)/operator:$(OPERATOR_VERSION) .

docker-build-bff: ## Build bff Docker image
	docker build -f tools/docker/Dockerfile.bff -t $(REGISTRY)/bff:$(BFF_VERSION) .

docker-push: ## Push all Docker images
	docker push $(REGISTRY)/agent:$(AGENT_VERSION)
	docker push $(REGISTRY)/keeper:$(KEEPER_VERSION)
	docker push $(REGISTRY)/operator:$(OPERATOR_VERSION)
	docker push $(REGISTRY)/bff:$(BFF_VERSION)

## —— Helm (app) —————————————————————————————————————————

helm-template: ## Render application Helm templates locally
	helm template $(HELM_RELEASE) $(HELM_CHART)

helm-install: ## Install/upgrade application Helm release
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) --wait

helm-uninstall: ## Uninstall application Helm release
	helm uninstall $(HELM_RELEASE)

## —— Helm (infra) ———————————————————————————————————————

helm-infra-deps: ## Download infrastructure chart dependencies
	helm dependency update $(HELM_INFRA_CHART)

helm-infra-lint: ## Lint the infrastructure Helm chart
	helm lint $(HELM_INFRA_CHART)

helm-infra-template: ## Render infrastructure Helm templates locally
	helm template $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART)

helm-infra-install: ## Install/upgrade infrastructure Helm release
	helm upgrade --install $(HELM_INFRA_RELEASE) $(HELM_INFRA_CHART) --wait

helm-infra-uninstall: ## Uninstall infrastructure Helm release
	helm uninstall $(HELM_INFRA_RELEASE)

## —— CI —————————————————————————————————————————————————

ci: lint test test-integration test-contract build docker-build ## Full CI pipeline

## —— Utilities ——————————————————————————————————————————

clean: ## Clean build artifacts
	rm -rf bin/

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
