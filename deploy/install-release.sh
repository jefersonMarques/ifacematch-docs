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


release_banner="$("$binary" -version 2>&1)" || {
    echo "[BLOCK] Cannot execute candidate documentation binary" >&2
    exit 2
}
if [[ "$release_banner" != *"$release_version"* ]]; then
    echo "[BLOCK] Candidate binary does not match requested version $release_version" >&2
    exit 2
fi

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
    local status="${1:-1}"
    trap - ERR

    if [[ -f "$backup" ]]; then
        if install -m 0755 "$backup" "$destination" && \
            systemctl restart ifacematch-docs.service; then
            echo "[ROLLBACK] Previous documentation binary restored: $backup" >&2
        else
            echo "[CRITICAL] Automatic rollback failed; recover from: $backup" >&2
        fi
    fi

    rm -f -- "$candidate"
    exit "$status"
}

wait_for_deployment() {
    local attempt
    for ((attempt = 1; attempt <= 30; attempt++)); do
        if ! systemctl is-active --quiet ifacematch-docs.service; then
            sleep 1
            continue
        fi

        if ! curl --fail --silent --show-error --max-time 2 \
            http://127.0.0.1:8091/healthz >/dev/null 2>&1; then
            sleep 1
            continue
        fi

        if [[ "$release_version" == "v0.1.2" ]]; then
            if ! curl --fail --silent --show-error --max-time 3 \
                http://127.0.0.1:8091/upload-image 2>/dev/null | \
                grep -Fq '/v1/enrollments/images'; then
                sleep 1
                continue
            fi

            if ! curl --fail --silent --show-error --max-time 3 \
                http://127.0.0.1:8091/openapi-preview.json 2>/dev/null | \
                grep -Fq '"/v1/enrollments/images"'; then
                sleep 1
                continue
            fi
        fi
        return 0
    done
    return 1
}

trap 'rollback $?' ERR
install -m 0755 "$destination" "$backup"
install -m 0755 "$binary" "$candidate"
mv -f -- "$candidate" "$destination"

systemctl restart ifacematch-docs.service
if ! wait_for_deployment; then
    echo "[BLOCK] Documentation service failed readiness checks; restoring previous binary" >&2
    rollback 1
fi

trap - ERR
echo "[OK] Updated iFaceMatch Docs to $release_version"
echo "[BACKUP] $backup"
