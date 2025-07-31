# Makefile for kubectl-kaito plugin

# Version and build information
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build configuration
BINARY_NAME = kubectl-kaito
BINARY_PATH = bin/$(BINARY_NAME)
PKG = github.com/kaito-project/kubectl-kaito
CMD_PKG = ./cmd/kubectl-kaito
LDFLAGS = -ldflags "-X ${PKG}/pkg/cmd.version=${VERSION} -X ${PKG}/pkg/cmd.commit=${COMMIT} -X ${PKG}/pkg/cmd.date=${DATE}"

GOLANGCI_LINT_VER := latest
GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(abspath $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN)-$(GOLANGCI_LINT_VER))


# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	mkdir -p bin
	go build ${LDFLAGS} -o ${BINARY_PATH} ${CMD_PKG}

# Build for multiple platforms
.PHONY: build-all
build-all:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 ${CMD_PKG}
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-arm64 ${CMD_PKG}
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 ${CMD_PKG}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 ${CMD_PKG}
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe ${CMD_PKG}

# Run unit tests with race detection and coverage report
.PHONY: unit-tests
unit-tests:
	@echo "Running unit tests with race detection and coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./cmd/... ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total

# Run e2e tests
.PHONY: test-e2e
test-e2e: test-e2e-kind
	@echo "E2E tests completed (excluding AKS tests)"

# Run e2e tests with Kind cluster
.PHONY: test-e2e-kind
test-e2e-kind:
	@echo "Setting up Kind cluster and running e2e tests..."
	./e2e/setup-kind.sh
	cd e2e && go test -v -timeout=15m -run "TestKindClusterOperations"
	./e2e/cleanup-kind.sh

# Run e2e tests with AKS cluster (creates billable resources)
.PHONY: test-e2e-aks
test-e2e-aks:
	@echo "Warning: This will create billable Azure resources!"
	@echo "Setting up AKS cluster and running e2e tests..."
	./e2e/setup-aks.sh
	cd e2e && go test -v -timeout=30m -run "TestAKSClusterOperations"
	@echo "💡 To clean up AKS resources, run: ./e2e/cleanup-aks.sh"

# Setup Kind cluster for manual testing
.PHONY: setup-kind
setup-kind:
	./e2e/setup-kind.sh

# Setup AKS cluster for manual testing
.PHONY: setup-aks
setup-aks:
	./e2e/setup-aks.sh

# Cleanup Kind cluster
.PHONY: cleanup-kind
cleanup-kind:
	./e2e/cleanup-kind.sh

# Cleanup AKS cluster
.PHONY: cleanup-aks
cleanup-aks:
	./e2e/cleanup-aks.sh

# Lint the code

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download and install golangci-lint locally.

.PHONY: ginkgo
ginkgo: $(GOLANGCI_LINT) ## Download and install ginkgo locally.

$(GOLANGCI_LINT): ## Download and install golangci-lint locally.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/golangci/golangci-lint/cmd/golangci-lint $(GOLANGCI_LINT_BIN) $(GOLANGCI_LINT_VER)


.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run -v


# Format the code
.PHONY: fmt
fmt:
	go fmt ./...

# Vet the code
.PHONY: vet
vet:
	go vet ./...

# Tidy dependencies
.PHONY: tidy
tidy:
	go mod tidy

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out coverage.html

# Install the plugin locally
.PHONY: install
install: build
	cp ${BINARY_PATH} /usr/local/bin/

# Uninstall the plugin
.PHONY: uninstall
uninstall:
	rm -f /usr/local/bin/${BINARY_NAME}

# Create release archives
.PHONY: release
release: build-all
	mkdir -p dist/archives
	cd dist && tar -czf archives/${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz ${BINARY_NAME}-linux-amd64
	cd dist && tar -czf archives/${BINARY_NAME}-${VERSION}-linux-arm64.tar.gz ${BINARY_NAME}-linux-arm64
	cd dist && tar -czf archives/${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz ${BINARY_NAME}-darwin-amd64
	cd dist && tar -czf archives/${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz ${BINARY_NAME}-darwin-arm64
	cd dist && zip archives/${BINARY_NAME}-${VERSION}-windows-amd64.zip ${BINARY_NAME}-windows-amd64.exe

# Generate checksums for release
.PHONY: checksums
checksums:
	cd dist/archives && sha256sum * > checksums.sha256

# Development setup
.PHONY: dev-setup
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Quick check before commit
.PHONY: check
check: fmt vet lint unit-tests pre-commit-run-staged

# Create Krew plugin manifest
.PHONY: krew-manifest
krew-manifest:
	@echo "Creating Krew plugin manifest..."
	@cat krew/kaito.yaml.template | sed 's/{{VERSION}}/${VERSION}/g' > krew/kaito.yaml

# Validate Krew manifest
.PHONY: krew-validate
krew-validate:
	kubectl krew install --manifest=krew/kaito.yaml

# Run the binary (for testing)
.PHONY: run
run: build
	${BINARY_PATH}

# GoReleaser targets
.PHONY: goreleaser-init goreleaser-check goreleaser-build
goreleaser-init:
	goreleaser init

goreleaser-check:
	goreleaser check

goreleaser-build:
	goreleaser build --snapshot --rm-dist

# Pre-commit targets
.PHONY: pre-commit-install
pre-commit-install:
	@echo "Installing pre-commit..."
	pip install pre-commit
	pre-commit install --install-hooks
	@echo "Pre-commit installed successfully!"

.PHONY: pre-commit-update
pre-commit-update:
	@echo "Updating pre-commit hooks..."
	pre-commit autoupdate

.PHONY: pre-commit-run
pre-commit-run:
	@echo "Running pre-commit on all files..."
	pre-commit run --all-files

.PHONY: pre-commit-run-staged
pre-commit-run-staged:
	@echo "Running pre-commit on staged files..."
	pre-commit run

.PHONY: pre-commit-clean
pre-commit-clean:
	@echo "Cleaning pre-commit cache..."
	pre-commit clean

.PHONY: pre-commit-uninstall
pre-commit-uninstall:
	@echo "Uninstalling pre-commit hooks..."
	pre-commit uninstall

# Security check target (required by pre-commit config)
.PHONY: security-check
security-check:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Git hooks setup
.PHONY: setup-git-hooks
setup-git-hooks: pre-commit-install
	@echo "Setting up git hooks for automatic pre-commit execution..."
	@echo '#!/bin/bash' > .git/hooks/pre-push
	@echo 'echo "Running pre-commit checks before push..."' >> .git/hooks/pre-push
	@echo 'make pre-commit-run-staged' >> .git/hooks/pre-push
	@echo 'if [ $$? -ne 0 ]; then' >> .git/hooks/pre-push
	@echo '  echo "Pre-commit checks failed. Push aborted."' >> .git/hooks/pre-push
	@echo '  exit 1' >> .git/hooks/pre-push
	@echo 'fi' >> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "Git pre-push hook installed successfully!"

# Complete setup for development
.PHONY: dev-setup-complete
dev-setup-complete: dev-setup setup-git-hooks
	@echo "Development environment setup complete!"
	@echo "Pre-commit will now run automatically on git push."

# Update help target to include new targets
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary to bin/"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  unit-tests   - Run unit tests with race detection and coverage"
	@echo "  test-e2e     - Run e2e tests (Kind cluster only)"
	@echo "  test-e2e-kind - Setup Kind cluster, run tests, and cleanup"
	@echo "  test-e2e-aks - Setup AKS cluster and run tests (billable)"
	@echo "  setup-kind   - Setup Kind cluster for manual testing"
	@echo "  setup-aks    - Setup AKS cluster for manual testing (billable)"
	@echo "  cleanup-kind - Cleanup Kind cluster"
	@echo "  cleanup-aks  - Cleanup AKS cluster"
	@echo "  lint         - Lint the code"
	@echo "  fmt          - Format the code"
	@echo "  vet          - Vet the code"
	@echo "  tidy         - Tidy dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install plugin locally"
	@echo "  uninstall    - Uninstall plugin"
	@echo "  release      - Create release archives"
	@echo "  checksums    - Generate checksums"
	@echo "  dev-setup    - Setup development tools"
	@echo "  dev-setup-complete - Complete development setup with pre-commit"
	@echo "  check        - Run all checks (fmt, vet, lint, unit-tests, pre-commit)"
	@echo "  krew-manifest - Create Krew manifest"
	@echo "  krew-validate - Validate Krew manifest"
	@echo "  run          - Build and run the binary"
	@echo "  goreleaser-init - Initialize GoReleaser configuration"
	@echo "  goreleaser-check - Check GoReleaser configuration"
	@echo "  goreleaser-build - Build with GoReleaser (snapshot)"
	@echo "  pre-commit-install - Install pre-commit hooks"
	@echo "  pre-commit-update - Update pre-commit hooks"
	@echo "  pre-commit-run - Run pre-commit on all files"
	@echo "  pre-commit-run-staged - Run pre-commit on staged files"
	@echo "  pre-commit-clean - Clean pre-commit cache"
	@echo "  pre-commit-uninstall - Uninstall pre-commit hooks"
	@echo "  setup-git-hooks - Setup git hooks for automatic pre-commit"
	@echo "  security-check - Run security checks" 