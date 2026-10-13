BINARY         := comics
VERSION        := $(shell cat VERSION)
VERSION_TAG    := v$(VERSION)
BUILD_SHA      := $(shell git describe --always --dirty=.dirty --exclude='*' 2>/dev/null)
LDFLAGS        := -X main.version=$(VERSION) -X main.commit=$(BUILD_SHA)
BUILD_DIR      ?= build
OUTPUT         ?= $(BUILD_DIR)/$(BINARY)
RELEASE_BINARY := $(BINARY)-$(VERSION_TAG)

.DEFAULT_GOAL := test

.PHONY: help build release deploy frontend backend dev-backend dev-frontend test clean

help: ## Show this help.
	@awk 'BEGIN { FS = ":.*## " } /^[a-zA-Z_-]+:.*## / { printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

test: ## Run the Go tests and type-check the frontend (default).
	go test ./...
	cd web && npm run typecheck

build: frontend backend ## Build the frontend and embed it in build/comics.

release: ## Clean build of build/comics-vX.Y.Z, after vetting and testing.
	@echo "$(VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "VERSION must be X.Y.Z, got '$(VERSION)'"; exit 1; }
	$(MAKE) clean
	$(MAKE) frontend
	go vet ./...
	go test ./...
	$(MAKE) backend OUTPUT=$(BUILD_DIR)/$(RELEASE_BINARY)

deploy: ## Build a release binary and deploy it to HOST (e.g. `make deploy HOST=user@host`).
	@if [ -z "$(HOST)" ]; then \
		echo "Usage: make deploy HOST=<hostname>" >&2; \
		exit 1; \
	fi
	$(MAKE) release
	scp $(BUILD_DIR)/$(RELEASE_BINARY) $(HOST):.local/bin/$(RELEASE_BINARY)-$(BUILD_SHA)
	ssh $(HOST) 'ln -sf ~/.local/bin/$(RELEASE_BINARY)-$(BUILD_SHA) ~/.local/bin/$(BINARY) && systemctl --user restart $(BINARY)'

frontend: ## Install frontend dependencies and build web/dist.
	cd web && npm ci && npm run build

backend: ## Build the Go binary with the current web/dist embedded.
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(OUTPUT) ./cmd/comics

# Run these in two terminals; Vite proxies /api to the Go server on :3000.
dev-backend: ## Run the Go server on :3000 for development.
	go run -ldflags "$(LDFLAGS)" ./cmd/comics

dev-frontend: ## Run the Vite dev server with hot reload on :5173.
	cd web && npm run dev

clean: ## Remove build/ and the built frontend.
	rm -rf $(BUILD_DIR)
	find web/dist -mindepth 1 ! -name .keep -delete
