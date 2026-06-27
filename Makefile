# Qlxion Monorepo Makefile
# ============================================

# Variables
GATEWAY_DIR := api-gateway
AUTH_DIR := services/auth-service
DEPLOY_DIR := deploy

# Docker
DOCKER_REGISTRY ?= qlxion
GATEWAY_IMAGE := $(DOCKER_REGISTRY)/api-gateway
AUTH_IMAGE := $(DOCKER_REGISTRY)/auth-service
TAG ?= latest

# Go
GO := go
GOTEST := $(GO) test
GOVET := $(GO) vet
GOBUILD := $(GO) build
GOCLEAN := $(GO) clean

# Colors
BLUE := \033[36m
GREEN := \033[32m
RED := \033[31m
YELLOW := \033[33m
RESET := \033[0m

.PHONY: all help build test clean docker-build docker-up docker-down k8s-deploy k8s-delete migrate lint proto

# ============================================
# Default Target
# ============================================
all: build

# ============================================
# Help
# ============================================
help: ## Show this help message
	@echo "$(BLUE)Qlxion Monorepo - Available Commands:$(RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'

# ============================================
# Build
# ============================================
build: build-gateway build-auth ## Build all services

build-gateway: ## Build API Gateway
	@echo "$(BLUE)Building API Gateway...$(RESET)"
	cd $(GATEWAY_DIR) && $(GOBUILD) -o bin/gateway cmd/gateway/main.go
	@echo "$(GREEN)API Gateway built successfully$(RESET)"

build-auth: ## Build Auth Service
	@echo "$(BLUE)Building Auth Service...$(RESET)"
	cd $(AUTH_DIR) && $(GOBUILD) -o bin/auth cmd/auth/main.go
	@echo "$(GREEN)Auth Service built successfully$(RESET)"

# ============================================
# Development
# ============================================
dev-gateway: ## Run API Gateway in development mode
	@echo "$(BLUE)Starting API Gateway (dev)...$(RESET)"
	cd $(GATEWAY_DIR) && JWT_SECRET=dev-secret go run cmd/gateway/main.go

dev-auth: ## Run Auth Service in development mode
	@echo "$(BLUE)Starting Auth Service (dev)...$(RESET)"
	cd $(AUTH_DIR) && JWT_SECRET=dev-secret go run cmd/auth/main.go

dev-all: ## Run all services with docker-compose
	@echo "$(BLUE)Starting all services...$(RESET)"
	cd $(DEPLOY_DIR) && docker-compose up --build

# ============================================
# Testing
# ============================================
test: test-gateway test-auth ## Run all tests

test-gateway: ## Run API Gateway tests
	@echo "$(BLUE)Testing API Gateway...$(RESET)"
	cd $(GATEWAY_DIR) && $(GOTEST) -v ./...

test-auth: ## Run Auth Service tests
	@echo "$(BLUE)Testing Auth Service...$(RESET)"
	cd $(AUTH_DIR) && $(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(RESET)"
	cd $(GATEWAY_DIR) && $(GOTEST) -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	cd $(AUTH_DIR) && $(GOTEST) -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html

# ============================================
# Linting
# ============================================
lint: ## Run linter on all code
	@echo "$(BLUE)Running linter...$(RESET)"
	cd $(GATEWAY_DIR) && $(GOVET) ./...
	cd $(AUTH_DIR) && $(GOVET) ./...
	@echo "$(GREEN)Linting complete$(RESET)"

fmt: ## Format all Go code
	@echo "$(BLUE)Formatting code...$(RESET)"
	find . -name "*.go" -not -path "*/vendor/*" -exec gofmt -w {} \;
	@echo "$(GREEN)Formatting complete$(RESET)"

# ============================================
# Dependencies
# ============================================
deps: ## Download all dependencies
	@echo "$(BLUE)Downloading dependencies...$(RESET)"
	cd $(GATEWAY_DIR) && go mod download && go mod tidy
	cd $(AUTH_DIR) && go mod download && go mod tidy
	@echo "$(GREEN)Dependencies downloaded$(RESET)"

deps-update: ## Update all dependencies
	@echo "$(BLUE)Updating dependencies...$(RESET)"
	cd $(GATEWAY_DIR) && go get -u ./... && go mod tidy
	cd $(AUTH_DIR) && go get -u ./... && go mod tidy
	@echo "$(GREEN)Dependencies updated$(RESET)"

# ============================================
# Docker
# ============================================
docker-build: docker-build-gateway docker-build-auth ## Build all Docker images

docker-build-gateway: ## Build API Gateway Docker image
	@echo "$(BLUE)Building API Gateway Docker image...$(RESET)"
	docker build -t $(GATEWAY_IMAGE):$(TAG) -f $(GATEWAY_DIR)/Dockerfile .
	@echo "$(GREEN)API Gateway image built: $(GATEWAY_IMAGE):$(TAG)$(RESET)"

docker-build-auth: ## Build Auth Service Docker image
	@echo "$(BLUE)Building Auth Service Docker image...$(RESET)"
	docker build -t $(AUTH_IMAGE):$(TAG) -f $(AUTH_DIR)/Dockerfile .
	@echo "$(GREEN)Auth Service image built: $(AUTH_IMAGE):$(TAG)$(RESET)"

docker-push: ## Push all Docker images
	@echo "$(BLUE)Pushing Docker images...$(RESET)"
	docker push $(GATEWAY_IMAGE):$(TAG)
	docker push $(AUTH_IMAGE):$(TAG)
	@echo "$(GREEN)Images pushed$(RESET)"

docker-up: ## Start services with docker-compose
	@echo "$(BLUE)Starting services with docker-compose...$(RESET)"
	cd $(DEPLOY_DIR) && docker-compose up -d

docker-down: ## Stop services with docker-compose
	@echo "$(BLUE)Stopping services...$(RESET)"
	cd $(DEPLOY_DIR) && docker-compose down

docker-logs: ## View docker-compose logs
	cd $(DEPLOY_DIR) && docker-compose logs -f

docker-clean: ## Remove all containers and volumes
	@echo "$(RED)Removing all containers and volumes...$(RESET)"
	cd $(DEPLOY_DIR) && docker-compose down -v --remove-orphans

# ============================================
# Database
# ============================================
migrate-up: ## Run database migrations
	@echo "$(BLUE)Running database migrations...$(RESET)"
	cd $(AUTH_DIR) && go run cmd/migrate/main.go up

migrate-down: ## Rollback database migrations
	@echo "$(YELLOW)Rolling back database migrations...$(RESET)"
	cd $(AUTH_DIR) && go run cmd/migrate/main.go down

migrate-create: ## Create a new migration (usage: make migrate-create name=add_users_table)
	@echo "$(BLUE)Creating migration: $(name)...$(RESET)"
	@touch $(AUTH_DIR)/migrations/$$(date +%Y%m%d%H%M%S)_$(name).up.sql
	@touch $(AUTH_DIR)/migrations/$$(date +%Y%m%d%H%M%S)_$(name).down.sql
	@echo "$(GREEN)Migration files created$(RESET)"

db-seed: ## Seed the database with initial data
	@echo "$(BLUE)Seeding database...$(RESET)"
	psql -h localhost -U postgres -d auth_db -f $(AUTH_DIR)/migrations/001_init.up.sql

# ============================================
# Kubernetes
# ============================================
k8s-deploy: ## Deploy to Kubernetes
	@echo "$(BLUE)Deploying to Kubernetes...$(RESET)"
	kubectl apply -f $(DEPLOY_DIR)/k8s/shared/namespace.yaml
	kubectl apply -f $(DEPLOY_DIR)/k8s/shared/secrets.yaml
	kubectl apply -f $(DEPLOY_DIR)/k8s/api-gateway/
	kubectl apply -f $(DEPLOY_DIR)/k8s/auth-service/
	@echo "$(GREEN)Deployment complete$(RESET)"

k8s-delete: ## Remove from Kubernetes
	@echo "$(RED)Removing from Kubernetes...$(RESET)"
	kubectl delete -f $(DEPLOY_DIR)/k8s/api-gateway/
	kubectl delete -f $(DEPLOY_DIR)/k8s/auth-service/
	@echo "$(GREEN)Cleanup complete$(RESET)"

k8s-status: ## Check Kubernetes status
	@echo "$(BLUE)Kubernetes Status:$(RESET)"
	kubectl get pods -n qlxion
	kubectl get svc -n qlxion
	kubectl get ingress -n qlxion

k8s-logs-gateway: ## View API Gateway logs
	kubectl logs -f deployment/api-gateway -n qlxion

k8s-logs-auth: ## View Auth Service logs
	kubectl logs -f deployment/auth-service -n qlxion

# ============================================
# Protobuf / gRPC
# ============================================
proto: ## Generate Go code from protobuf files
	@echo "$(BLUE)Generating protobuf code...$(RESET)"
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto
	@echo "$(GREEN)Protobuf code generated$(RESET)"

# ============================================
# Utilities
# ============================================
clean: ## Clean build artifacts
	@echo "$(RED)Cleaning build artifacts...$(RESET)"
	rm -rf $(GATEWAY_DIR)/bin $(AUTH_DIR)/bin
	$(GOCLEAN)
	@echo "$(GREEN)Clean complete$(RESET)"

swagger: ## Generate Swagger documentation
	@echo "$(BLUE)Generating Swagger docs...$(RESET)"
	swag init -g $(GATEWAY_DIR)/cmd/gateway/main.go -o api-docs/
	@echo "$(GREEN)Swagger docs generated$(RESET)"

security-scan: ## Run security scan
	@echo "$(BLUE)Running security scan...$(RESET)"
	cd $(GATEWAY_DIR) && gosec ./...
	cd $(AUTH_DIR) && gosec ./...

.PHONY: ci

ci: lint test build ## Run CI pipeline locally
	@echo "$(GREEN)CI pipeline complete$(RESET)"
