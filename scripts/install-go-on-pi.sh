#!/usr/bin/env bash

set -euo pipefail

GO_VERSION="${GO_VERSION:-$(wget -qO- https://go.dev/VERSION?m=text)}"
ARCH="$(uname -m)"

case "$ARCH" in
  aarch64|arm64)
    GO_ARCHIVE="${GO_VERSION}.linux-arm64.tar.gz"
    ;;
  armv7l|armv7|armv6l|armhf)
    GO_ARCHIVE="${GO_VERSION}.linux-armv6l.tar.gz"
    ;;
  *)
    echo "unsupported Raspberry Pi architecture: ${ARCH}" >&2
    echo "supported: aarch64/arm64 and 32-bit armv7l/armv6l Raspberry Pi OS" >&2
    exit 1
    ;;
esac

DOWNLOAD_URL="https://go.dev/dl/${GO_ARCHIVE}"

echo "Installing ${GO_VERSION} for ${ARCH}"
echo "Archive: ${GO_ARCHIVE}"

wget "${DOWNLOAD_URL}"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "${GO_ARCHIVE}"

PROFILE_FILE="${HOME}/.profile"
PATH_LINE='export PATH=$PATH:/usr/local/go/bin'

if ! grep -Fqx "${PATH_LINE}" "${PROFILE_FILE}" 2>/dev/null; then
  printf '\n%s\n' "${PATH_LINE}" >> "${PROFILE_FILE}"
fi

echo
echo "Go installed to /usr/local/go"
echo "Reload your shell or run:"
echo "  source ~/.profile"
echo "Then verify with:"
echo "  go version"
