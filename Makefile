include .env


.PHONY: clean prepare kill reset restart dev-gateway dev-catalog dev-cart dev-order dev-payment dev-identity install-golang-migrate migration-version migration migrate-up migrate-down migrate-reset migrate-force proto-clean proto-gen docker-build-seq docker-up docker-up-d docker-rebuild docker-down docker-reset docker-logs

# --- DYNAMIC DATABASE MIGRATION STRATEGY ---

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),migration-version migration migrate-up migrate-down migrate-reset migrate-force))
ifndef service
$(error Missing required parameter. Usage: 'make <target> service=<catalog|cart|order|payment|identity>')
endif
endif

ifeq ($(service),catalog)
    TARGET_DB_URL = postgres://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_CATALOG_DB_NAME)?sslmode=disable
    MIGRATION_PATH = ./migrations/catalog
else ifeq ($(service),cart)
    TARGET_DB_URL = postgres://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_CART_DB_NAME)?sslmode=disable
    MIGRATION_PATH = ./migrations/cart
else ifeq ($(service),order)
    TARGET_DB_URL = postgres://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_ORDER_DB_NAME)?sslmode=disable
    MIGRATION_PATH = ./migrations/order
else ifeq ($(service),payment)
    TARGET_DB_URL = postgres://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_PAYMENT_DB_NAME)?sslmode=disable
    MIGRATION_PATH = ./migrations/payment
else ifeq ($(service),identity)
    TARGET_DB_URL = postgres://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_IDENTITY_DB_NAME)?sslmode=disable
    MIGRATION_PATH = ./migrations/identity
endif


clean:
	@echo "Cleaning build artifacts..."
	@rm -f bin/*.exe bin/*.exe~ bin/*.log
	@rm -rf bin/gateway-state bin/catalog-state bin/cart-state bin/order-state bin/payment-state bin/identity-state

prepare: clean
	@mkdir -p bin/gateway-state
	@mkdir -p bin/catalog-state
	@mkdir -p bin/cart-state
	@mkdir -p bin/order-state
	@mkdir -p bin/payment-state
	@mkdir -p bin/identity-state

kill:
	@echo "Force killing all running service processes..."
	-@taskkill /F /IM api-gateway.exe 2>/dev/null || true
	-@taskkill /F /IM catalog-service.exe 2>/dev/null || true
	-@taskkill /F /IM cart-service.exe 2>/dev/null || true
	-@taskkill /F /IM order-service.exe 2>/dev/null || true
	-@taskkill /F /IM payment-service.exe 2>/dev/null || true
	-@taskkill /F /IM identity-service.exe 2>/dev/null || true
	-@taskkill /F /IM air.exe 2>/dev/null || true
	@echo "Releasing gRPC ports..."
	-@powershell -NoProfile -ExecutionPolicy Bypass -File scripts/kill-ports.ps1
	@echo "Done."

# Kill hanging processes then clean artifacts
reset: kill clean prepare
	@echo "Waiting for ports to be released..."
	@sleep 3

# Full fresh restart
restart: kill dev

dev-gateway:
	air -c .air.gateway.toml

dev-catalog:
	air -c .air.catalog.toml

dev-cart:
	air -c .air.cart.toml

dev-order:
	air -c .air.order.toml

dev-payment:
	air -c .air.payment.toml

dev-identity:
	air -c .air.identity.toml

# Starts the clean, synchronized live-reload environment
dev: reset
	@make -j 6 dev-gateway dev-catalog dev-cart dev-order dev-payment dev-identity

# ── MIGRATION COMMAND CHANNELS ────────────────────────────────────────────────

install-golang-migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migration-version:
	migrate -database "$(TARGET_DB_URL)" -path $(MIGRATION_PATH) version

migration:
	migrate create -ext sql -dir $(MIGRATION_PATH) -seq $(name)

migrate-up:
	migrate -database "$(TARGET_DB_URL)" -path $(MIGRATION_PATH) up

migrate-down:
	migrate -database "$(TARGET_DB_URL)" -path $(MIGRATION_PATH) down

migrate-reset:
	migrate -database "$(TARGET_DB_URL)" -path $(MIGRATION_PATH) down -all
	migrate -database "$(TARGET_DB_URL)" -path $(MIGRATION_PATH) up

migrate-force:
	migrate -database "$(TARGET_DB_URL)" -path $(MIGRATION_PATH) force $(version)


PROTO_DIR     = proto/$(service)
PROTO_GEN_OUT = proto/$(service)

proto-clean:
ifndef service
	$(error Missing required parameter. Usage: 'make proto-gen service=<service-name>')
endif
	@echo "Cleaning old generated grpc code in [$(PROTO_GEN_OUT)]..."
	@if [ -d "$(PROTO_GEN_OUT)" ]; then rm -f $(PROTO_GEN_OUT)/*.pb.go; fi

proto-gen: proto-clean
	@echo "Generating Go grpc source files for [$(service)]..."
	@mkdir -p $(PROTO_GEN_OUT)
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=. --go_opt=module=github.com/mafi020/ecom-golang-micro \
		--go-grpc_out=. --go-grpc_opt=module=github.com/mafi020/ecom-golang-micro \
		$(PROTO_DIR)/*.proto
	@echo "Successfully compiled Protobuf contracts into [$(PROTO_GEN_OUT)]!"


# Docker

# ── DOCKER LIFECYCLE COMMANDS ─────────────────────────────────────────────────

DOCKER_SERVICES = identity-service catalog-service cart-service order-service payment-service api-gateway

# Build each service image one at a time — avoids 6 parallel Go compiles
# fighting over the same CPU/RAM during the first build.
docker-build-seq:
	@for svc in $(DOCKER_SERVICES); do \
		echo "==> Building $$svc..."; \
		docker compose build $$svc || exit 1; \
	done
	@echo "All service images built successfully."

# Start the stack from already-built images (no rebuild).
docker-up:
	docker compose up

# Start the stack detached.
docker-up-d:
	docker compose up -d

# One-shot replacement for 'docker compose up --build' that won't hammer your CPU.
docker-rebuild: docker-build-seq docker-up

# Stop and remove containers, keep volumes (Postgres/RabbitMQ data survives).
docker-down:
	docker compose down

# Full reset: stop containers AND wipe volumes — fresh databases next run.
docker-reset:
	docker compose down -v

# Tail logs for everything, or one service: make docker-logs svc=order-service
docker-logs:
	docker compose logs -f $(svc)