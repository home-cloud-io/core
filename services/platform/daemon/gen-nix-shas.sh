#!/bin/bash

VERSION=$(git rev-parse HEAD)
REVISION=$(git rev-parse HEAD)

echo "version: $REVISION"
sed -e "s/__VERSION__/$VERSION/g" -e "s/__REVISION__/$REVISION/g" default.nix.tmpl > tmp/default.nix

cd tmp/

# We first try to build and it fails with hash mismatch, and we use it to populate sha256.
SRC_SHA256="$(nix-build 2>&1 | grep -oP 'got:\s+\Ksha256-\S+')"
echo "src    : $SRC_SHA256"
sed -i -e "s|hash = \"\";|hash = \"$SRC_SHA256\";|g" default.nix

# We try again to build and it fails with hash mismatch, and we use it to populate vendorSha256.
VENDOR_SHA256="$(nix-build 2>&1 | grep -oP 'got:\s+\Ksha256-\S+')"
echo "vendor : $VENDOR_SHA256"
sed -i -e "s|vendorHash = \"\";|vendorHash = \"$VENDOR_SHA256\";|g" default.nix
