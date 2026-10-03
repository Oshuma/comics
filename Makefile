VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_DIR ?= build
BINARY    ?= $(BUILD_DIR)/comics

.PHONY: build frontend backend dev-backend dev-frontend test clean

build: frontend backend

frontend:
	cd web && npm ci && npm run build

backend:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/comics

# Run these in two terminals; Vite proxies /api to the Go server on :3000.
dev-backend:
	go run ./cmd/comics

dev-frontend:
	cd web && npm run dev

test:
	go test ./...
	cd web && npm run typecheck

clean:
	rm -rf $(BUILD_DIR)
	find web/dist -mindepth 1 ! -name .keep -delete
