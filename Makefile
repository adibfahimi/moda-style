# Moda Style — developer Makefile.
#
# A thin, dependency-free wrapper around the handful of commands a contributor
# needs day to day. Run `make` (or `make help`) to see the available targets.

# All Go modules that make up the workspace. `go test ./...` cannot cross module
# boundaries from the repository root, so targets iterate over these explicitly.
GO_MODULES   := common services/auth-service services/product-service services/cart-service services/order-service services/admin-service
FRONTEND_DIR := frontend

.DEFAULT_GOAL := help
.PHONY: help test test-common test-services test-cover test-frontend typecheck-frontend \
        install-frontend build-frontend check fmt vet tidy build \
        run-auth run-product run-cart run-order run-admin up down logs clean

## help: list available targets
help:
	@echo "Moda Style — available make targets:"; \
	grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | sort

## test: run every Go test across all workspace modules
test: test-common test-services

test-common:
	@echo ">> go test ./common/..."; \
	cd common && go test ./...

test-services:
	@for m in $(filter services/%,$(GO_MODULES)); do \
		echo ">> go test ./$$m/..."; \
		(cd $$m && go test ./...) || exit 1; \
	done

## test-cover: run Go tests with a coverage summary per module
test-cover:
	@for m in $(GO_MODULES); do \
		echo ">> coverage for $$m"; \
		(cd $$m && go test -cover ./...) || exit 1; \
	done

## test-frontend: run the Vitest suite for the SPA
test-frontend:
	cd $(FRONTEND_DIR) && bun run test

## typecheck-frontend: type-check the SPA with tsc
typecheck-frontend:
	cd $(FRONTEND_DIR) && bun run typecheck

## build-frontend: produce the production SPA bundle into frontend/dist
build-frontend:
	cd $(FRONTEND_DIR) && bun run build

## install-frontend: install SPA dependencies from the committed bun.lock
install-frontend:
	cd $(FRONTEND_DIR) && bun install --frozen-lockfile

## check: run everything CI runs (vet, all Go tests, frontend types and tests)
check: vet test test-frontend typecheck-frontend

## fmt: format all Go source with gofmt
fmt:
	@for m in $(GO_MODULES); do (cd $$m && gofmt -w .); done

## vet: run go vet across all workspace modules
vet:
	@for m in $(GO_MODULES); do echo ">> go vet ./$$m/..."; (cd $$m && go vet ./...) || exit 1; done

## tidy: sync go.mod/go.sum for every module and the workspace
tidy:
	@for m in $(GO_MODULES); do (cd $$m && go mod tidy); done
	go work sync

## build: compile every service binary into tmp/
build:
	@mkdir -p tmp
	@for m in $(filter services/%,$(GO_MODULES)); do \
		name=$$(basename $$m); \
		echo ">> building $$name"; \
		(cd $$m && go build -o ../../tmp/$$name .) || exit 1; \
	done

## run-auth: run the auth service locally (requires env vars)
run-auth:
	cd services/auth-service && go run .

## run-product: run the product service locally (requires env vars)
run-product:
	cd services/product-service && go run .

## run-cart: run the cart service locally (requires env vars)
run-cart:
	cd services/cart-service && go run .

## run-order: run the order service locally (requires env vars)
run-order:
	cd services/order-service && go run .

## run-admin: run the admin service locally (requires env vars)
run-admin:
	cd services/admin-service && go run .

## up: start the full stack with Docker Compose
up:
	docker compose up --build

## down: stop the stack and remove the database volume
down:
	docker compose down -v

## logs: follow logs from the whole stack
logs:
	docker compose logs -f

## clean: remove build artifacts
clean:
	rm -rf tmp
	cd $(FRONTEND_DIR) && rm -rf dist

