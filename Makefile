VERSION   := $(shell cat VERSION)
COMMIT    := $(shell git describe --always --dirty=.dirty --exclude='*' 2>/dev/null)
LDFLAGS   := -X main.version=$(VERSION) -X main.commit=$(COMMIT)
BUILD_DIR ?= build
BINARY    ?= $(BUILD_DIR)/comics

.PHONY: build release frontend backend dev-backend dev-frontend test clean

build: frontend backend

# Clean build of a stripped, static binary, after vetting and testing.
release:
	@echo "$(VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "VERSION must be X.Y.Z, got '$(VERSION)'"; exit 1; }
	$(MAKE) clean
	$(MAKE) frontend
	go vet ./...
	go test ./...
	$(MAKE) backend BINARY=$(BUILD_DIR)/comics-v$(VERSION)

frontend:
	cd web && npm ci && npm run build

backend:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w $(LDFLAGS)" -o $(BINARY) ./cmd/comics

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
