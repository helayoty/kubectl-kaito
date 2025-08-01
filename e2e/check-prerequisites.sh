#!/bin/bash

# Copyright (c) 2024 Kaito Project
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# check-prerequisites.sh - Check if all required tools for e2e tests are available

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo "🔍 Checking prerequisites for kaito-kubectl-plugin e2e tests..."
echo ""

# Track overall status
ALL_GOOD=true

# Check Go
echo -n "Checking Go installation... "
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}✓${NC} Found Go ${GO_VERSION}"
else
    echo -e "${RED}✗${NC} Go not found"
    echo "  Install: https://golang.org/doc/install"
    ALL_GOOD=false
fi

# Check kubectl
echo -n "Checking kubectl... "
if command -v kubectl &> /dev/null; then
    KUBECTL_VERSION=$(kubectl version --client --short 2>/dev/null | awk '{print $3}' || echo "unknown")
    echo -e "${GREEN}✓${NC} Found kubectl ${KUBECTL_VERSION}"
else
    echo -e "${RED}✗${NC} kubectl not found"
    echo "  Install: https://kubernetes.io/docs/tasks/tools/"
    ALL_GOOD=false
fi

# Check Docker
echo -n "Checking Docker... "
if command -v docker &> /dev/null; then
    if docker info &> /dev/null; then
        DOCKER_VERSION=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "unknown")
        echo -e "${GREEN}✓${NC} Found Docker ${DOCKER_VERSION} (running)"
    else
        echo -e "${YELLOW}⚠${NC} Docker found but not running"
        echo "  Start Docker Desktop or Docker daemon"
    fi
else
    echo -e "${RED}✗${NC} Docker not found"
    echo "  Install: https://docs.docker.com/get-docker/"
    ALL_GOOD=false
fi

# Check Kind
echo -n "Checking Kind... "
if command -v kind &> /dev/null; then
    KIND_VERSION=$(kind version | awk '{print $2}' 2>/dev/null || echo "unknown")
    echo -e "${GREEN}✓${NC} Found Kind ${KIND_VERSION}"
else
    echo -e "${YELLOW}⚠${NC} Kind not found (required for Kind cluster tests)"
    echo "  Install: go install sigs.k8s.io/kind@latest"
    echo "  Or skip Kind tests with: SKIP_KIND_TESTS=true"
fi

# Check Azure CLI
echo -n "Checking Azure CLI... "
if command -v az &> /dev/null; then
    AZ_VERSION=$(az version --query '"azure-cli"' -o tsv 2>/dev/null || echo "unknown")
    echo -e "${GREEN}✓${NC} Found Azure CLI ${AZ_VERSION}"
    
    # Check Azure authentication
    echo -n "Checking Azure authentication... "
    if az account show &> /dev/null; then
        ACCOUNT_NAME=$(az account show --query 'name' -o tsv 2>/dev/null || echo "unknown")
        echo -e "${GREEN}✓${NC} Authenticated (${ACCOUNT_NAME})"
    else
        echo -e "${YELLOW}⚠${NC} Not authenticated"
        echo "  Run: az login"
        echo "  Or skip AKS tests with: SKIP_AKS_TESTS=true"
    fi
else
    echo -e "${YELLOW}⚠${NC} Azure CLI not found (required for AKS tests)"
    echo "  Install: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    echo "  Or skip AKS tests with: SKIP_AKS_TESTS=true"
fi

echo ""

# Check disk space
echo -n "Checking available disk space... "
if command -v df &> /dev/null; then
    AVAILABLE_GB=$(df / | awk 'NR==2 {printf "%.1f", $4/1024/1024}')
    if (( $(echo "$AVAILABLE_GB > 10" | bc -l) )); then
        echo -e "${GREEN}✓${NC} ${AVAILABLE_GB}GB available"
    else
        echo -e "${YELLOW}⚠${NC} Only ${AVAILABLE_GB}GB available (recommend 10GB+)"
    fi
else
    echo -e "${YELLOW}⚠${NC} Cannot check disk space"
fi

# Show environment variables that can be used
echo ""
echo "📝 Environment variables for test customization:"
echo "  SKIP_KIND_TESTS=true         # Skip Kind cluster tests"
echo "  SKIP_AKS_TESTS=true          # Skip AKS cluster tests"
echo "  KIND_CLUSTER_NAME=my-cluster # Custom Kind cluster name"
echo "  AKS_RESOURCE_GROUP=my-rg     # Custom AKS resource group"
echo "  AKS_LOCATION=eastus          # Custom AKS location"
echo ""

# Show test commands
echo "🚀 Available test commands:"
echo "  make test-e2e-basic          # Basic tests (no cluster)"
echo "  make test-e2e-kind           # Kind cluster tests"
echo "  make test-e2e-aks            # AKS cluster tests (billable)"
echo "  make test-e2e                # Basic + Kind tests"
echo "  make test-e2e-all            # All tests including AKS"
echo ""

# Cost warning for AKS
echo "💰 AKS Cost Information:"
echo "  - Control plane: ~$0.10/hour"
echo "  - Standard_NC6s_v3 node: ~$0.90/hour"
echo "  - Estimated 30-min test cost: ~$0.50-1.00"
echo "  - Tests automatically clean up resources"
echo ""

# Final status
if [ "$ALL_GOOD" = true ]; then
    echo -e "${GREEN}✅ All core prerequisites are available!${NC}"
    echo "You can run: make test-e2e-basic"
    exit 0
else
    echo -e "${RED}❌ Some required tools are missing.${NC}"
    echo "Install missing tools and run this script again."
    exit 1
fi