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

# setup-kind.sh - Setup Kind cluster and prerequisites for e2e tests
#
# Environment variables:
#   KAITO_IMAGE_TAG - Kaito image tag to use (default: 0.3.0)
#
# Usage:
#   ./setup-kind.sh                    # Uses default tag
#   KAITO_IMAGE_TAG=0.4.0 ./setup-kind.sh  # Uses specific tag

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

KIND_CLUSTER_NAME="kaito-e2e-kind"
TIMEOUT_CLUSTER=10m
TIMEOUT_INSTALL=5m
KAITO_IMAGE_TAG="${KAITO_IMAGE_TAG:-0.3.0}"

echo -e "${BLUE}🚀 Setting up Kind cluster and prerequisites for e2e tests${NC}"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install kubectl
install_kubectl() {
    echo -e "${YELLOW}📦 Installing kubectl...${NC}"
    
    if command_exists brew; then
        brew install kubectl
    elif command_exists curl; then
        curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
        chmod +x kubectl
        sudo mv kubectl /usr/local/bin/
    else
        echo -e "${RED}❌ Unable to install kubectl: no suitable package manager found${NC}"
        exit 1
    fi
}

# Function to install helm
install_helm() {
    echo -e "${YELLOW}📦 Installing helm...${NC}"
    
    if command_exists brew; then
        brew install helm
    elif command_exists curl; then
        curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    else
        echo -e "${RED}❌ Unable to install helm: no suitable package manager found${NC}"
        exit 1
    fi
}

# Function to install kind
install_kind() {
    echo -e "${YELLOW}📦 Installing kind...${NC}"
    
    if command_exists brew; then
        brew install kind
    elif command_exists go; then
        go install sigs.k8s.io/kind@latest
    else
        echo -e "${RED}❌ Unable to install kind: no suitable installation method found${NC}"
        exit 1
    fi
}

# Function to install Docker
install_docker() {
    echo -e "${YELLOW}📦 Docker not running. Please install and start Docker Desktop${NC}"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo -e "${BLUE}💡 For macOS: Download Docker Desktop from https://www.docker.com/products/docker-desktop${NC}"
    else
        echo -e "${BLUE}💡 For Linux: Follow instructions at https://docs.docker.com/engine/install/${NC}"
    fi
    exit 1
}

# Check and install prerequisites
echo -e "${BLUE}🔍 Checking prerequisites...${NC}"

# Check kubectl
if ! command_exists kubectl; then
    install_kubectl
else
    echo -e "${GREEN}✅ kubectl is available${NC}"
fi

# Check helm
if ! command_exists helm; then
    install_helm
else
    echo -e "${GREEN}✅ helm is available${NC}"
fi

# Check kind
if ! command_exists kind; then
    install_kind
else
    echo -e "${GREEN}✅ kind is available${NC}"
fi

# Check Docker
if ! docker info >/dev/null 2>&1; then
    install_docker
else
    echo -e "${GREEN}✅ Docker is running${NC}"
fi

# Clean up any existing cluster
echo -e "${BLUE}🧹 Cleaning up any existing Kind cluster...${NC}"
if kind get clusters | grep -q "^${KIND_CLUSTER_NAME}$"; then
    echo -e "${YELLOW}⚠️  Deleting existing cluster: ${KIND_CLUSTER_NAME}${NC}"
    kind delete cluster --name "${KIND_CLUSTER_NAME}"
fi

# Create Kind cluster
echo -e "${BLUE}🏗️  Creating Kind cluster: ${KIND_CLUSTER_NAME}${NC}"

cat <<EOF > /tmp/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${KIND_CLUSTER_NAME}
nodes:
- role: control-plane
- role: worker
  extraMounts:
  - hostPath: /var/run/docker.sock
    containerPath: /var/run/docker.sock
EOF

timeout "${TIMEOUT_CLUSTER}" kind create cluster --config /tmp/kind-config.yaml

echo -e "${BLUE}⏳ Waiting for cluster to be ready...${NC}"
timeout "${TIMEOUT_CLUSTER}" kubectl wait --for=condition=Ready nodes --all --timeout=600s

# Install Kaito helm chart
echo -e "${BLUE}📦 Installing Kaito helm chart...${NC}"

# Set kubectl context
kubectl config use-context "kind-${KIND_CLUSTER_NAME}"

# Add Kaito helm repository following Kaito project approach
echo -e "${BLUE}📦 Adding Kaito helm repository...${NC}"
helm repo add kaito https://azure.github.io/kaito
helm repo update

# Create kaito-system namespace
echo -e "${BLUE}🔧 Creating kaito-system namespace...${NC}"
kubectl create namespace kaito-system --dry-run=client -o yaml | kubectl apply -f -

# Install CRDs first (following Kaito pattern)
echo -e "${BLUE}📋 Installing Kaito CRDs...${NC}"
kubectl apply -f https://raw.githubusercontent.com/kaito-project/kaito/main/charts/kaito/workspace-crd/crds/kaito.sh_workspaces.yaml

# Install Kaito operator with specific values for Kind
echo -e "${BLUE}🚀 Installing Kaito operator (tag: ${KAITO_IMAGE_TAG})...${NC}"
timeout "${TIMEOUT_INSTALL}" helm install kaito kaito/kaito \
    --namespace kaito-system \
    --set image.repository=mcr.microsoft.com/aks/kaito/workspace \
    --set image.tag="${KAITO_IMAGE_TAG}" \
    --set nodeProvisioner.enabled=false \
    --wait \
    --timeout 300s

echo -e "${BLUE}⏳ Waiting for Kaito operator to be ready...${NC}"
kubectl wait --for=condition=Available deployment/kaito-controller-manager \
    --namespace kaito-system \
    --timeout=300s

# Deploy nginx for testing
echo -e "${BLUE}📦 Deploying nginx for testing...${NC}"
kubectl create deployment nginx --image=nginx:latest
kubectl expose deployment nginx --port=80 --target-port=80

echo -e "${BLUE}⏳ Waiting for nginx to be ready...${NC}"
kubectl wait --for=condition=Available deployment/nginx --timeout=300s

echo -e "${GREEN}✅ Kind cluster setup complete!${NC}"
echo -e "${GREEN}   Cluster: ${KIND_CLUSTER_NAME}${NC}"
echo -e "${GREEN}   Context: kind-${KIND_CLUSTER_NAME}${NC}"
echo -e "${GREEN}   Kaito operator: Installed and ready${NC}"
echo -e "${GREEN}   Test deployment: nginx ready${NC}"

# Clean up temp files
rm -f /tmp/kind-config.yaml

echo -e "${BLUE}🎯 Ready to run Kind e2e tests!${NC}" 