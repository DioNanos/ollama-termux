#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

echo "==> Verifying tracked files for public-release safety"

fail=0

check_glob() {
    local glob="$1"
    local label="$2"
    local matches
    matches="$(git ls-files "$glob")"
    if [ -n "$matches" ]; then
        echo "ERROR: tracked ${label} files found:"
        echo "$matches"
        fail=1
    fi
}

check_glob '*.pem' 'PEM'
check_glob '*.key' 'private key'
check_glob '*.p12' 'certificate bundle'
check_glob '*.pfx' 'certificate bundle'
check_glob '.env' 'dotenv'
check_glob '.env.*' 'dotenv'

secret_pattern='(-----BEGIN (RSA|EC|DSA|OPENSSH|PGP) PRIVATE KEY-----|gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,}|sk_live_[A-Za-z0-9]{16,}|sk-[A-Za-z0-9]{20,}|AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_-]{35}|xox[baprs]-[A-Za-z0-9-]{10,})'

if git ls-files -z | xargs -0 rg -nI --color=never -e "$secret_pattern" >/tmp/ollama-termux-public-safety.out 2>/dev/null; then
    echo "ERROR: potential secret-like content found in tracked files:"
    cat /tmp/ollama-termux-public-safety.out
    fail=1
fi
rm -f /tmp/ollama-termux-public-safety.out

if [ "$fail" -ne 0 ]; then
    exit 1
fi

echo "Public-release safety checks passed."
