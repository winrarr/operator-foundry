#!/usr/bin/env bash
set -euo pipefail

destination=${1:?destination is required}
version=${HUBBLE_VERSION:?HUBBLE_VERSION is required}
os=${HUBBLE_OS:-$(go env GOOS)}
arch=${HUBBLE_ARCH:-$(go env GOARCH)}

case "${os}/${arch}" in
  linux/amd64|linux/arm64|darwin/amd64|darwin/arm64|windows/amd64|windows/arm64) ;;
  *)
    echo "unsupported Hubble platform: ${os}/${arch}" >&2
    exit 1
    ;;
esac

asset="hubble-${os}-${arch}.tar.gz"
base_url="https://github.com/cilium/hubble/releases/download/${version}"
temporary_directory=$(mktemp -d)
cleanup() {
  find -P "${temporary_directory}" -mindepth 1 -delete
  rmdir "${temporary_directory}"
}
trap cleanup EXIT

curl --fail --silent --show-error --location "${base_url}/${asset}" --output "${temporary_directory}/${asset}"
curl --fail --silent --show-error --location "${base_url}/${asset}.sha256sum" --output "${temporary_directory}/checksum"
(
  cd "${temporary_directory}"
  sha256sum --check checksum
)

tar --extract --gzip --file "${temporary_directory}/${asset}" --directory "${temporary_directory}"
install -m 0755 "${temporary_directory}/hubble" "${destination}"
