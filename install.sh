#!/bin/bash
set -euo pipefail

if [ -z "${1:-}" ]; then
    echo "Usage: $0 <platform>"
    echo "  Platforms: darwin-amd64, darwin-arm64, linux-amd64, linux-arm, linux-arm64, windows-amd64"
    exit 1
fi

PLATFORM="$1"
REPO="esenmx/SpoofDPI-Turkiye"

TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$TAG" ]; then
    echo "Error: could not resolve latest release tag for ${REPO}"
    exit 1
fi

ARCHIVE="spoofdpi-${PLATFORM}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

echo "Downloading ${URL}"
curl -fsSL -O "${URL}"

mkdir -p ~/.spoofdpi/bin

tar -xzf "./${ARCHIVE}"
rm -f "./${ARCHIVE}"
mv ./spoofdpi ~/.spoofdpi/bin/

echo
echo "SpoofDPI-Turkiye ${TAG} (${PLATFORM}) başarıyla indirildi."
echo "Lütfen rcfile(.bashrc or .zshrc etc..) dosyanızın altına şunu ekleyin:"
echo
echo ">>    export PATH=\$PATH:~/.spoofdpi/bin"
