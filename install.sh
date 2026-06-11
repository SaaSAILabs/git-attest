#!/bin/sh
set -e

# Detect OS and Architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
    Linux*)     OS="linux";;
    Darwin*)    OS="darwin";;
    CYGWIN*|MINGW*|MSYS*) OS="windows";;
    *)          echo "Unsupported OS: ${OS}"; exit 1;;
esac

case "${ARCH}" in
    x86_64*)    ARCH="x86_64";;
    i386*)      ARCH="i386";;
    arm64*|aarch64*) ARCH="arm64";;
    *)          echo "Unsupported Architecture: ${ARCH}"; exit 1;;
esac

echo "=> Detected ${OS} ${ARCH}"

# Fetch the latest release metadata from GitHub
echo "=> Fetching latest release version..."
LATEST_URL="https://api.github.com/repos/SaaSAILabs/git-attest/releases/latest"
VERSION=$(curl -s $LATEST_URL | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "Error: Could not fetch latest release."
    exit 1
fi

echo "=> Latest version is $VERSION"

# Construct download URL based on OS and Architecture
# GoReleaser naming template: git-attest_Darwin_arm64.tar.gz / git-attest_Linux_x86_64.tar.gz
TARBALL="git-attest_$(echo $OS | awk '{print toupper(substr($0,1,1))substr($0,2)})_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/SaaSAILabs/git-attest/releases/download/${VERSION}/${TARBALL}"

echo "=> Downloading $DOWNLOAD_URL"

# Create temporary directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Download and extract
curl -sSL -o "$TARBALL" "$DOWNLOAD_URL"
tar -xzf "$TARBALL"

# Install binary
echo "=> Installing to /usr/local/bin (may require sudo)"
sudo mv git-attest /usr/local/bin/

# Clean up
cd - >/dev/null
rm -rf "$TMP_DIR"

# Initialize git-attest
echo "=> Initializing global git-attest hook"
git attest init

echo "=> Installation complete!"
