#!/bin/sh
# Build the exact asset names consumed by hostlier package install --version TAG.
set -eu
cd "$(dirname "$0")/.."
output="$PWD/dist/releases"
mkdir -p "$output"
manifest=$(mktemp "$output/.SHA256SUMS.XXXXXX")
trap 'rm -f "$manifest"' EXIT HUP INT TERM
python3 scripts/catalog-project-packages.py --verify
packages="postgres redis neo4j aiven railway terraform package minion harnest sandbox-browser sandbox-computer"
if [ "${WITH_PULUMI:-0}" = 1 ]; then packages="$packages pulumi"; fi
for target_os in ${RELEASE_OS:-linux darwin}; do
  for target_arch in ${RELEASE_ARCH:-amd64 arm64}; do
    for name in $packages; do
      artifact="hostlier-$name-$target_os-$target_arch"
      if [ "$name" = minion ] || [ "$name" = harnest ] || [ "$name" = sandbox-browser ] || [ "$name" = sandbox-computer ]; then
        cp "binaries/$artifact" "$output/$artifact"
      elif [ "$name" = pulumi ]; then
        (cd packages/pulumi && CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags='-s -w' -o "$output/$artifact" ./cmd/hostlier-pulumi)
      else
        CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags='-s -w' -o "$output/$artifact" "./cmd/hostlier-$name"
      fi
      (cd "$output" && if command -v sha256sum >/dev/null 2>&1; then sha256sum "$artifact"; else shasum -a 256 "$artifact"; fi) >> "$manifest"
    done
  done
done
mv "$manifest" "$output/SHA256SUMS"
printf 'Release assets: %s\n' "$output"
