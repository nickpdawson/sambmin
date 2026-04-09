# Sambmin Makefile
# Build targets for FreeBSD, Linux, and macOS

BINARY    = sambmin
API_DIR   = api
WEB_DIR   = web
DIST_DIR  = dist
CMD_PKG   = ./cmd/sambmin

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   = -X main.Version=$(VERSION) -X main.CommitSHA=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

PLATFORMS = freebsd/amd64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: build build-all frontend test clean dist help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build for current platform
	cd $(API_DIR) && go build -ldflags '$(LDFLAGS)' -o ../$(BINARY) $(CMD_PKG)
	@echo "Built $(BINARY) $(VERSION)"

build-all: ## Cross-compile for all platforms
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		sh -c "cd $(API_DIR) && go build -ldflags '$(LDFLAGS)' -o ../$(DIST_DIR)/$(BINARY)-$${platform%/*}-$${platform#*/} $(CMD_PKG)" && \
		echo "  Built $(BINARY)-$${platform%/*}-$${platform#*/}" || exit 1; \
	done

frontend: ## Build React frontend
	cd $(WEB_DIR) && npm install && npm run build

test: ## Run Go tests
	cd $(API_DIR) && go test ./...

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf $(DIST_DIR)

dist: clean build-all frontend ## Build release tarballs for all platforms
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		name=$(BINARY)-$(VERSION)-$${platform%/*}-$${platform#*/}; \
		mkdir -p $(DIST_DIR)/$$name; \
		cp $(DIST_DIR)/$(BINARY)-$${platform%/*}-$${platform#*/} $(DIST_DIR)/$$name/$(BINARY); \
		cp -r $(WEB_DIR)/dist $(DIST_DIR)/$$name/web; \
		cp $(API_DIR)/config.example.yaml $(DIST_DIR)/$$name/config.example.yaml; \
		cp -r scripts $(DIST_DIR)/$$name/scripts 2>/dev/null || true; \
		cp LICENSE $(DIST_DIR)/$$name/ 2>/dev/null || true; \
		tar -czf $(DIST_DIR)/$$name.tar.gz -C $(DIST_DIR) $$name; \
		rm -rf $(DIST_DIR)/$$name; \
		echo "  Packaged $$name.tar.gz"; \
	done
	@echo "Release artifacts in $(DIST_DIR)/"
