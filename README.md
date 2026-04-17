# Ollama Termux

> Built from upstream [Ollama](https://github.com/ollama/ollama), adapted as a
> Termux-first fork for Android ARM64 devices.

[![npm](https://img.shields.io/npm/v/@mmmbuto/ollama-termux?style=flat-square&logo=npm)](https://www.npmjs.com/package/@mmmbuto/ollama-termux)
[![platform](https://img.shields.io/badge/platform-Android%20ARM64-3DDC84?style=flat-square&logo=android&logoColor=white)](https://termux.dev)
[![license](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](./LICENSE)

---

## What This Is

`ollama-termux` is an explicit fork of upstream Ollama for Termux on modern
Android ARM64 phones. It keeps the upstream Ollama codebase and release naming
scheme, but adds a Termux-specific distribution flow and mobile-oriented runtime
behavior.

### What We Keep From Upstream

- Upstream Ollama source tree and MIT license
- Upstream version lineage, published as `v<upstream>-termux.N`
- Standard `ollama` CLI and server behavior where Termux-specific changes are
  not required

### What This Fork Changes

- Keeps only the launcher integrations we actually support on Termux:
  **Claude Code** and **Codex**
- Uses `termux-open-url` when browser/OAuth flows need to open URLs on Android
- Tunes CPU thread selection, memory heuristics, flash attention defaults, and
  context limits for modern phones
- Ships prebuilt Android ARM64 release assets through GitHub Releases and
  installs them through the npm package

### What This Fork Does Not Claim

- This is **not** a minimal patch port like `codex-termux`
- This is **not** a drop-in replacement for upstream Ollama on desktop/server
  systems
- This does **not** add GPU inference today

---

## Termux-Specific Behavior

The current fork-only behavior is intentional and user-visible:

- Launcher integrations restricted to Claude Code + Codex
- Claude launched with `--dangerously-skip-permissions`
- Codex launched with `--dangerously-bypass-approvals-and-sandbox`
- Android-aware memory heuristic based on `MemTotal`
- Big-core detection via `/sys/devices/system/cpu/.../cpufreq`
- Context ladder capped for mobile RAM tiers
- Flash attention enabled automatically on CPU-only mobile targets

These are fork behaviors, not upstream behaviors.

---

## Requirements

- Android device running Termux
- ARM64 CPU
- Node.js `>=18`
- Enough free space for the downloaded release tarball and extracted libraries

This package is intended for **Termux on Android ARM64 only**.

---

## Installation

### 1. Install the fork

```bash
pkg update && pkg upgrade -y
pkg install nodejs-lts -y

npm install -g @mmmbuto/ollama-termux
```

The npm package is an installer wrapper. During `postinstall`, it downloads the
matching GitHub Release asset for the published package version, verifies the
SHA256 checksum when available, and installs:

- `bin/ollama`
- `lib/ollama/*.so`

### 2. Install supported coding CLIs

```bash
# Claude Code
npm install -g @anthropic-ai/claude-code

# Codex Termux fork
npm install -g @mmmbuto/codex-cli-termux
```

### 3. Verify

```bash
ollama --version
ollama serve
```

Links:

- npm: https://www.npmjs.com/package/@mmmbuto/ollama-termux
- Releases: https://github.com/DioNanos/ollama-termux/releases
- Upstream: https://github.com/ollama/ollama

---

## Quickstart

```bash
# Start Ollama
ollama serve &

# Pull the recommended local models
ollama pull qwen3.5
ollama pull gemma4

# Launch Claude Code through Ollama with qwen3.5
ollama launch claude --model qwen3.5

# Launch Codex through Ollama with gemma4
ollama launch codex --model gemma4
```

---

## Build And Release

### Build locally on Linux

```bash
export NDK_ROOT=~/android-ndk/android-ndk-r27c
./scripts/build_termux.sh
```

Expected output:

- `dist/ollama-termux-<version>-android-arm64.tar.gz`
- `dist/ollama-termux-<version>-android-arm64.tar.gz.sha256`

### Release contract

Every published npm version must have a matching GitHub Release with these two
assets:

- `ollama-termux-<version>-android-arm64.tar.gz`
- `ollama-termux-<version>-android-arm64.tar.gz.sha256`

The installer depends on that contract. If the release asset is missing, the
npm package is not considered publish-ready.

See [docs/BUILDING.md](./docs/BUILDING.md) for the full cross-build flow.

---

## Device Notes

This fork is tuned for recent ARM64 phones such as:

- Pixel 9 Pro / Tensor G4
- Pixel 10 Pro / Tensor G5
- Galaxy S24+ / Snapdragon 8 Gen 3
- Galaxy S25 Ultra / Snapdragon 8 Elite

The shipped CPU backends target:

- `armv8.0` fallback
- `armv8.2 + dotprod + fp16`
- `armv8.6 + dotprod + fp16 + i8mm + sve2`

Backend selection is done at runtime by the ggml layer.

---

## Validation Status

Packaging checks completed in this repo before publish prep:

- `node -c install.js`
- `npm pack --dry-run`
- release asset naming aligned with installer and build script

Still required before public publish:

- GitHub Release asset generation for the tagged version
- Termux device install verification against the released asset

---

## License

This project is distributed under the MIT license.

- Original upstream work: [ollama/ollama](https://github.com/ollama/ollama)
- Termux fork work: Davide A. Gugliemi
