#!/bin/bash

# Script to prepare kubectl-kaito for Krew submission
# Usage: ./scripts/prepare-krew-release.sh v0.1.0

set -e

VERSION=${1:-}
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

# Validate version format
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format vX.Y.Z (e.g., v0.1.0)"
    exit 1
fi

REPO_OWNER="kaito-project"
REPO_NAME="kubectl-kaito"
BINARY_NAME="kubectl-kaito"

echo "🚀 Preparing Krew release for $VERSION"

# Clean and build
echo "📦 Building binaries..."
make clean
make build-all

# Create release archives
echo "📂 Creating release archives..."
make release

# Generate checksums
echo "🔐 Generating checksums..."
make checksums

# Check if all required files exist
REQUIRED_FILES=(
    "dist/archives/${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz"
    "dist/archives/${BINARY_NAME}-${VERSION}-linux-arm64.tar.gz"
    "dist/archives/${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz"
    "dist/archives/${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz"
    "dist/archives/${BINARY_NAME}-${VERSION}-windows-amd64.zip"
    "dist/archives/checksums.sha256"
)

echo "✅ Checking required files..."
for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Missing required file: $file"
        exit 1
    fi
    echo "  ✓ $file"
done

# Extract checksums
echo "📝 Extracting checksums..."
cd dist/archives

LINUX_AMD64_SHA=$(grep "${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz" checksums.sha256 | cut -d' ' -f1)
LINUX_ARM64_SHA=$(grep "${BINARY_NAME}-${VERSION}-linux-arm64.tar.gz" checksums.sha256 | cut -d' ' -f1)
DARWIN_AMD64_SHA=$(grep "${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz" checksums.sha256 | cut -d' ' -f1)
DARWIN_ARM64_SHA=$(grep "${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz" checksums.sha256 | cut -d' ' -f1)
WINDOWS_AMD64_SHA=$(grep "${BINARY_NAME}-${VERSION}-windows-amd64.zip" checksums.sha256 | cut -d' ' -f1)

cd ../..

# Generate Krew manifest
echo "📄 Generating Krew manifest..."
cat > "krew/kaito-${VERSION}.yaml" << EOF
apiVersion: krew.googlecontainertools.github.com/v1alpha2
kind: Plugin
metadata:
  name: kaito
spec:
  version: ${VERSION}
  homepage: https://github.com/${REPO_OWNER}/${REPO_NAME}
  shortDescription: Manage AI/ML model inference and fine-tuning with Kaito
  description: |
    kubectl-kaito is a command-line tool for managing AI/ML model inference 
    and fine-tuning workloads using the Kubernetes AI Toolchain Operator (Kaito).

    This plugin simplifies the deployment, management, and monitoring of AI models
    in Kubernetes clusters through Kaito workspaces.

    Features:
    - Deploy AI models for inference with automatic GPU provisioning
    - Fine-tune models with custom datasets using QLora and LoRA techniques  
    - Monitor workspace status with real-time updates
    - View logs from inference and training workloads
    - Discover available model presets for various AI model families
    - Delete workspaces with proper cleanup of GPU resources

  caveats: |
    This plugin requires:
    - Kubernetes cluster with Kaito operator installed
    - kubectl configured to access your cluster
    - Sufficient Azure VM quota for GPU instances (when using Azure)

  platforms:
  - selector:
      matchLabels:
        os: linux
        arch: amd64
    uri: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz
    sha256: "${LINUX_AMD64_SHA}"
    bin: ${BINARY_NAME}
  - selector:
      matchLabels:
        os: linux
        arch: arm64
    uri: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${BINARY_NAME}-${VERSION}-linux-arm64.tar.gz
    sha256: "${LINUX_ARM64_SHA}"
    bin: ${BINARY_NAME}
  - selector:
      matchLabels:
        os: darwin
        arch: amd64
    uri: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz
    sha256: "${DARWIN_AMD64_SHA}"
    bin: ${BINARY_NAME}
  - selector:
      matchLabels:
        os: darwin
        arch: arm64
    uri: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz
    sha256: "${DARWIN_ARM64_SHA}"
    bin: ${BINARY_NAME}
  - selector:
      matchLabels:
        os: windows
        arch: amd64
    uri: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${BINARY_NAME}-${VERSION}-windows-amd64.zip
    sha256: "${WINDOWS_AMD64_SHA}"
    bin: ${BINARY_NAME}.exe
EOF

echo "✅ Krew manifest generated: krew/kaito-${VERSION}.yaml"

# Update the main manifest
cp "krew/kaito-${VERSION}.yaml" "krew/kaito.yaml"

echo ""
echo "🎉 Release preparation complete!"
echo ""
echo "📋 Next steps:"
echo "1. Create GitHub release:"
echo "   git tag ${VERSION}"
echo "   git push origin ${VERSION}"
echo ""
echo "2. Upload these files to GitHub release:"
for file in "${REQUIRED_FILES[@]}"; do
    if [[ "$file" != *"checksums.sha256" ]]; then
        echo "   - $file"
    fi
done
echo ""
echo "3. Test the Krew manifest:"
echo "   kubectl krew install --manifest=krew/kaito-${VERSION}.yaml"
echo "   kubectl kaito --help"
echo "   kubectl krew uninstall kaito"
echo ""
echo "4. Submit to Krew index:"
echo "   - Fork https://github.com/kubernetes-sigs/krew-index"
echo "   - Copy krew/kaito-${VERSION}.yaml to plugins/kaito.yaml"
echo "   - Create PR with title: 'Add kubectl-kaito plugin'"
echo ""
echo "📄 Files ready for release:"
ls -la dist/archives/ 