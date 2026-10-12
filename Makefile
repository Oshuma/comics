BINARY         := comics
VERSION        := $(shell cat VERSION)
VERSION_TAG    := v$(VERSION)
BUILD_SHA      := $(shell git describe --always --dirty=.dirty --exclude='*' 2>/dev/null)
LDFLAGS        := -X main.version=$(VERSION) -X main.commit=$(BUILD_SHA)
BUILD_DIR      ?= build
OUTPUT         ?= $(BUILD_DIR)/$(BINARY)
RELEASE_BINARY := $(BINARY)-$(VERSION_TAG)

.PHONY: build release deploy frontend backend dev-backend dev-frontend test clean

build: frontend backend

# Clean build of a stripped, static binary, after vetting and testing.
release:
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

frontend:
	cd web && npm ci && npm run build

backend:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(OUTPUT) ./cmd/comics

# Run these in two terminals; Vite proxies /api to the Go server on :3000.
dev-backend:
	go run -ldflags "$(LDFLAGS)" ./cmd/comics

dev-frontend:
	cd web && npm run dev

test:
	go test ./...
	cd web && npm run typecheck

clean:
	rm -rf $(BUILD_DIR)
	find web/dist -mindepth 1 ! -name .keep -delete
