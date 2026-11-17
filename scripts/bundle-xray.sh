#!/bin/sh
# Keep the bundled core compatible with the deployed sing-box clients.
set -eu

version=v26.6.27
case "${1:?architecture required}" in
    amd64) asset=Xray-linux-64.zip; sha=b3e5902d06d6282fe53cfa2fc426058b9aeaa429b2c812e20887cd47f26d08bf ;;
    386) asset=Xray-linux-32.zip; sha=a30a7bc601ca4be80675ab862cbeae4cadb0bc3242ffb687fb6865b67b8c4e8f ;;
    arm64) asset=Xray-linux-arm64-v8a.zip; sha=13a251379bea366c2cf10363ad71e75734193d401f26f518bf0c25e5c8f8c931 ;;
    s390x) asset=Xray-linux-s390x.zip; sha=be883d2b998b238421838fe29e8dd7b4b731e23def7712c3bbdac04e16319524 ;;
    *) echo "Unsupported Xray architecture: $1" >&2; exit 1 ;;
esac

output=${2:?output path required}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL --retry 3 --connect-timeout 15 -o "$tmp/xray.zip"     "https://github.com/XTLS/Xray-core/releases/download/$version/$asset"
printf '%s  %s\n' "$sha" "$tmp/xray.zip" | sha256sum -c -
unzip -pq "$tmp/xray.zip" xray > "$tmp/xray"
install -m 0755 "$tmp/xray" "$output"
