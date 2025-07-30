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

# run-tests.sh - Easy test runner for kubectl-kaito e2e tests

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Options:"
    echo "  basic     Run basic tests only (no cluster)"
    echo "  kind      Run Kind cluster tests"
    echo "  aks       Run AKS cluster tests (creates billable resources)"
    echo "  all       Run all tests including AKS"
    echo "  check     Check prerequisites"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 basic   # Quick tests, no cluster needed"
    echo "  $0 kind    # Local cluster tests with Kind"
    echo "  $0 aks     # Azure cloud tests (costs money)"
    echo ""
}

# Function to check prerequisites
check_prerequisites() {
    echo -e "${BLUE}🔍 Checking prerequisites...${NC}"
    if [ -f "$SCRIPT_DIR/check-prerequisites.sh" ]; then
        "$SCRIPT_DIR/check-prerequisites.sh"
    else
        echo -e "${RED}❌ Prerequisites check script not found${NC}"
        exit 1
    fi
}

# Function to run basic tests
run_basic_tests() {
    echo -e "${BLUE}🧪 Running basic e2e tests (no cluster required)...${NC}"
    cd "$PROJECT_ROOT"
    make test-e2e-basic
    echo -e "${GREEN}✅ Basic tests completed${NC}"
}

# Function to run Kind tests
run_kind_tests() {
    echo -e "${BLUE}🧪 Running Kind cluster e2e tests...${NC}"
    
    # Check if Kind is available
    if ! command -v kind &> /dev/null; then
        echo -e "${RED}❌ Kind not found. Install it first:${NC}"
        echo "go install sigs.k8s.io/kind@latest"
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        echo -e "${RED}❌ Docker is not running. Please start Docker and try again.${NC}"
        exit 1
    fi
    
    cd "$PROJECT_ROOT"
    make test-e2e-kind
    echo -e "${GREEN}✅ Kind tests completed${NC}"
}

# Function to run AKS tests
run_aks_tests() {
    echo -e "${YELLOW}⚠️  Warning: AKS tests will create billable Azure resources!${NC}"
    echo -e "${YELLOW}   Estimated cost: $0.50-1.00 for a 30-minute test run${NC}"
    echo ""
    
    # Check if Azure CLI is available and authenticated
    if ! command -v az &> /dev/null; then
        echo -e "${RED}❌ Azure CLI not found. Install it first:${NC}"
        echo "https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
        exit 1
    fi
    
    if ! az account show &> /dev/null; then
        echo -e "${RED}❌ Not authenticated with Azure. Run 'az login' first.${NC}"
        exit 1
    fi
    
    # Get confirmation unless in CI
    if [ -z "$CI" ]; then
        echo -n "Do you want to proceed? (y/N): "
        read -r response
        if [[ ! "$response" =~ ^[Yy]$ ]]; then
            echo "Cancelled."
            exit 0
        fi
    fi
    
    echo -e "${BLUE}🧪 Running AKS cluster e2e tests...${NC}"
    cd "$PROJECT_ROOT"
    make test-e2e-aks
    echo -e "${GREEN}✅ AKS tests completed${NC}"
}

# Function to run all tests
run_all_tests() {
    echo -e "${YELLOW}⚠️  Warning: This will create billable Azure resources!${NC}"
    echo ""
    
    # Get confirmation unless in CI
    if [ -z "$CI" ]; then
        echo -n "Do you want to run ALL tests including AKS? (y/N): "
        read -r response
        if [[ ! "$response" =~ ^[Yy]$ ]]; then
            echo "Cancelled."
            exit 0
        fi
    fi
    
    echo -e "${BLUE}🧪 Running all e2e tests...${NC}"
    cd "$PROJECT_ROOT"
    make test-e2e-all
    echo -e "${GREEN}✅ All tests completed${NC}"
}

# Main script logic
case "${1:-help}" in
    basic)
        run_basic_tests
        ;;
    kind)
        run_kind_tests
        ;;
    aks)
        run_aks_tests
        ;;
    all)
        run_all_tests
        ;;
    check)
        check_prerequisites
        ;;
    help|--help|-h)
        show_usage
        ;;
    *)
        echo -e "${RED}❌ Unknown option: $1${NC}"
        echo ""
        show_usage
        exit 1
        ;;
esac