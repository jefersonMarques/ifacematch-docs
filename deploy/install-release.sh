#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 3 ]]; then
    echo "Usage: install-release.sh BINARY SHA256SUMS VERSION" >&2
    exit 2
fi

binary="$(realpath -- "$1")"
checksums="$(realpath -- "$2")"
release_version="$3"

[[ "$release_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
    echo "[BLOCK] Invalid release version" >&2
    exit 2
}

[[ "$(id -u)" -eq 0 ]] || {
    echo "[BLOCK] Root privileges are required" >&2
    exit 2
}

[[ -f "$binary" && ! -L "$binary" && -f "$checksums" && ! -L "$checksums" ]] || {
    echo "[BLOCK] Release files must be regular files" >&2
    exit 2
}

archive_dir="$(dirname "$binary")"
binary_name="$(basename "$binary")"
[[ "$binary_name" == "ifacematch-docs-linux-amd64" ]] || {
    echo "[BLOCK] Unexpected binary filename" >&2
    exit 2
}

(
    cd "$archive_dir"
    sha256sum --check --status "$checksums"
) || {
    echo "[BLOCK] Release SHA-256 verification failed" >&2
    exit 2
}

destination="/opt/ifacematch-docs/bin/ifacematch-docs"
[[ -f "$destination" && ! -L "$destination" ]] || {
    echo "[BLOCK] Existing service binary required; see README for first install" >&2
    exit 2
}

systemctl cat ifacematch-docs.service >/dev/null || {
    echo "[BLOCK] systemd service is missing" >&2
    exit 2
}

candidate="${destination}.candidate"
backup_dir="/opt/ifacematch-docs/backups"
mkdir -p -m 0700 "$backup_dir"
backup="${backup_dir}/$(date -u +%Y%m%dT%H%M%SZ)-ifacematch-docs"

rollback() {
    local status="$?"
    trap - ERR

    if [[ -f "$backup" ]]; then
        install -m 0755 "$backup" "$destination"
        systemctl restart ifacematch-docs.service || true
    fi

    rm -f -- "$candidate"
    exit "$status"
}

trap rollback ERR
install -m 0755 "$destination" "$backup"
install -m 0755 "$binary" "$candidate"
mv -f -- "$candidate" "$destination"

systemctl restart ifacematch-docs.service
curl --fail --silent --show-error --max-time 10 http://127.0.0.1:8091/healthz >/dev/null

trap - ERR
echo "[OK] Updated iFaceMatch Docs to $release_version"
echo "[BACKUP] $backup"
