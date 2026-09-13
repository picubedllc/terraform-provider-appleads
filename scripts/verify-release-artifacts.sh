#!/usr/bin/env bash
# Verify GoReleaser produced Terraform Registry zip names for the required platforms.
set -euo pipefail

dist="${1:-dist}"
if [ ! -d "$dist" ]; then
  echo "missing $dist; run goreleaser first" >&2
  exit 1
fi

required=(
  darwin_arm64
  darwin_amd64
  linux_amd64
  linux_arm64
  windows_amd64
)

missing=0
for platform in "${required[@]}"; do
  # Snapshot versions look like 0.0.0-SNAPSHOT-*; real releases are X.Y.Z.
  match=$(find "$dist" -maxdepth 1 -name "terraform-provider-appleads_*_${platform}.zip" | head -n 1 || true)
  if [ -z "$match" ]; then
    echo "missing zip for ${platform}" >&2
    missing=1
  else
    echo "ok ${match##*/}"
  fi
done

if [ "$missing" -ne 0 ]; then
  echo "artifacts in ${dist}:" >&2
  ls -1 "$dist" >&2
  exit 1
fi
