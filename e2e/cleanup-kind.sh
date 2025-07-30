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

# cleanup-kind.sh - Cleanup Kind cluster after e2e tests

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

KIND_CLUSTER_NAME="kaito-e2e-kind"

echo -e "${BLUE}🧹 Cleaning up Kind cluster: ${KIND_CLUSTER_NAME}${NC}"

# Check if kind is available
if ! command -v kind >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Kind not found, cluster may already be cleaned up${NC}"
    exit 0
fi

# Check if cluster exists
if ! kind get clusters | grep -q "^${KIND_CLUSTER_NAME}$"; then
    echo -e "${GREEN}✅ Cluster ${KIND_CLUSTER_NAME} does not exist, nothing to clean up${NC}"
    exit 0
fi

# Delete the cluster
echo -e "${YELLOW}🗑️  Deleting Kind cluster: ${KIND_CLUSTER_NAME}${NC}"
kind delete cluster --name "${KIND_CLUSTER_NAME}"

echo -e "${GREEN}✅ Kind cluster cleanup complete!${NC}" 