#!/usr/bin/env bash
# Publish one release of an app to the update catalog (all-or-nothing).
#
# Usage:
#   publish-release.sh <app> <version> <os/arch>=<file> [<os/arch>=<file> ...]
#
# Environment:
#   CATALOG_URL           base URL, e.g. https://updates.r0ma.cloud
#   CATALOG_UPLOAD_TOKEN  the app's upload token (cat_up_...)
#   CATALOG_SIGNING_KEY   the app's Ed25519 private key (base64 32-byte seed)
#
# Requires: bash, curl, openssl >= 3.0, sha256sum, base64, xxd or od.
set -euo pipefail

die() { echo "publish-release: $*" >&2; exit 1; }

[ $# -ge 3 ] || die "usage: $0 <app> <version> <os/arch>=<file> ..."
: "${CATALOG_URL:?CATALOG_URL is required}"
: "${CATALOG_UPLOAD_TOKEN:?CATALOG_UPLOAD_TOKEN is required}"
: "${CATALOG_SIGNING_KEY:?CATALOG_SIGNING_KEY is required}"

app="$1"; version="${2#v}"; shift 2
api="${CATALOG_URL%/}/api/${app}"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

# Wrap the raw 32-byte seed in a PKCS#8 DER envelope so openssl can use it.
key="$workdir/key.pem"
{
  printf '\x30\x2e\x02\x01\x00\x30\x05\x06\x03\x2b\x65\x70\x04\x22\x04\x20'
  printf '%s' "$CATALOG_SIGNING_KEY" | base64 -d
} | openssl pkey -inform DER -out "$key" 2>/dev/null || die "CATALOG_SIGNING_KEY is not a valid base64 Ed25519 seed"
chmod 600 "$key"

call() { # call METHOD URL [curl args...] -> prints body, fails on HTTP >= 400
  local method="$1" url="$2"; shift 2
  local out="$workdir/resp" code
  code="$(curl -sS -o "$out" -w '%{http_code}' -X "$method" \
    -H "Authorization: Bearer ${CATALOG_UPLOAD_TOKEN}" "$@" "$url")"
  if [ "$code" -ge 400 ]; then
    echo "HTTP $code from $method $url: $(cat "$out")" >&2
    return 1
  fi
  cat "$out"
}

echo "Creating draft ${app} ${version}..."
call POST "${api}/releases" -H 'Content-Type: application/json' \
  --data "{\"version\":\"${version}\"}" >/dev/null

for spec in "$@"; do
  target="${spec%%=*}"; file="${spec#*=}"
  goos="${target%%/*}"; arch="${target#*/}"
  [ -f "$file" ] || die "file not found: $file"
  sha="$(sha256sum "$file" | cut -d' ' -f1)"
  msg="$workdir/msg"
  printf 'actualizaciones/v1\napp=%s\nversion=%s\nos=%s\narch=%s\nsha256=%s\n' \
    "$app" "$version" "$goos" "$arch" "$sha" > "$msg"
  sig="$(openssl pkeyutl -sign -inkey "$key" -rawin -in "$msg" | base64 -w0)"
  echo "Uploading ${target} (sha256 ${sha})..."
  call PUT "${api}/releases/${version}/artifacts/${goos}/${arch}" \
    -H "X-Signature: ${sig}" -H 'Content-Type: application/octet-stream' \
    --data-binary "@${file}" >/dev/null
done

echo "Finalizing ${app} ${version}..."
call POST "${api}/releases/${version}/finalize" >/dev/null
echo "Published ${app} ${version}."
