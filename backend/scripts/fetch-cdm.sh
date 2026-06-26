#!/bin/bash
set -e

# Default version if not specified
CDM_VERSION="${1:-7.0.0-dev.78}"
TARGET_DIR="backend/internal/cdm"
TEMP_DIR=$(mktemp -d)

echo "Fetching FINOS CDM Golang distribution version ${CDM_VERSION}..."

# Construct URL
URL="https://repo1.maven.org/maven2/org/finos/cdm/cdm-golang/${CDM_VERSION}/cdm-golang-${CDM_VERSION}.tar.gz"

# Download to temp dir
echo "Downloading from ${URL}..."
if ! curl -sfL "${URL}" -o "${TEMP_DIR}/cdm.tar.gz"; then
    echo "Error: Failed to download CDM distribution. Verify the version exists on Maven Central."
    rm -rf "${TEMP_DIR}"
    exit 1
fi

# Prepare target directory
echo "Preparing target directory ${TARGET_DIR}..."
mkdir -p "${TARGET_DIR}"
# Remove existing Go files to ensure clean state (preserving README and mock if needed, though usually cleaner to wipe)
# Using find to delete .go files but maybe better to clean specifically.
find "${TARGET_DIR}" -name "*.go" -type f -not -name "mock_cdm.go" -delete

# Extract
echo "Extracting..."
tar -xzf "${TEMP_DIR}/cdm.tar.gz" -C "${TARGET_DIR}" --strip-components=1

echo "Content of target dir:"
ls -F "${TARGET_DIR}"

# Cleanup
rm -rf "${TEMP_DIR}"

echo "Check if extraction worked..."
if [ -z "$(ls -A ${TARGET_DIR})" ]; then
   echo "Warning: Target directory is empty. Extraction might have failed or archive structure is unexpected."
   exit 1
fi

echo "Successfully placed CDM Go files in ${TARGET_DIR}"
echo "You can now run the generator:"
echo "  go run ./cmd/cdm-generator/main.go -path ./internal/cdm -output ./catalog.json"
