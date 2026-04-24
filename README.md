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
  **Codex** (primary), **Qwen Code** (secondary), and **Claude Code**
  (frozen @2.1.112)
- Uses `termux-open-url` when browser/OAuth flows need to open URLs on Android
- Tunes CPU thread selection, memory heuristics, flash attention defaults, and
  context limits for modern phones
- Ships prebuilt Android ARM64 release assets through GitHub Releases and
  installs them through the npm package

### What This Fork Does Not Claim

- This is **not** a minimal patch port like `codex-termux`
- This is **not** a drop-in replacement for upstream Ollama on desktop/server
  systems

### Experimental

- **Vulkan on Termux** (opt-in): the runner can route `dlopen("libvulkan.so")`
  through the Android system loader (`/system/lib64`), which has namespace
  access to the vendor GPU ICD at `/vendor/lib64/hw/vulkan.*.so` that Termux
  processes are normally blocked from. Verified on Pixel 9 Pro / Mali-G715.
  Build with `BUILD_VULKAN=1 ./scripts/build_termux.sh`, enable at runtime
  with `OLLAMA_VULKAN=1`. See
  [docs/VULKAN_ANDROID_LOADER.md](./docs/VULKAN_ANDROID_LOADER.md) for the
  loader mechanism, and [docs/BENCHMARKS.md](./docs/BENCHMARKS.md) for
  CPU-vs-Vulkan throughput on Pixel 9 Pro. Highlights on Mali-G715:
  `gemma4:e2b` 2.44 → **7.37 tok/s** (3.0×), `gemma4:e4b` 1.81 →
  **4.45 tok/s** (2.5×), `medgemma:latest` 2.43 → **3.96 tok/s** (1.6×).
  All models reach 100% layer offload.

---

## Termux-Specific Behavior

The current fork-only behavior is intentional and user-visible:

- Launcher integrations: Codex (primary), Qwen Code (secondary),
  Claude Code (frozen @2.1.112)
- Codex launched with `--dangerously-bypass-approvals-and-sandbox`
- Qwen launched with `--approval-mode yolo` and routed through the
  local Ollama OpenAI-compat `/v1` endpoint
- Claude launched with `--dangerously-skip-permissions`
- Android-aware memory heuristic based on `MemTotal`
- Big-core detection via `/sys/devices/system/cpu/.../cpufreq`
- Context ladder capped for mobile RAM tiers
- Flash attention enabled automatically on CPU-only mobile targets
- Runner prepends `/system/lib64` to `LD_LIBRARY_PATH` so an optional
  `ggml-vulkan` backend can reach the Android vendor GPU driver

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

Launcher priority on Termux is **Codex (primary) → Qwen (secondary) → Claude (frozen)**.

```bash
# Codex — primary (our Termux fork)
npm install -g @mmmbuto/codex-cli-termux

# Qwen Code — secondary (our Termux fork)
npm install -g @mmmbuto/qwen-code-termux

# Claude Code — frozen at 2.1.112 (Anthropic dropped Termux after)
npm install -g @anthropic-ai/claude-code@2.1.112
```

Important:

- **Codex** is the recommended coding agent on Termux. Fork ships
  Android-tuned runtime and is published at `@mmmbuto/codex-cli-termux`.
- **Qwen Code** is the secondary agent, also a Termux-native fork
  (`@mmmbuto/qwen-code-termux`). Routes through Ollama via the
  OpenAI-compatible endpoint.
- **Claude Code** is **frozen** at `2.1.112` — the last version that
  still ships native Termux support. `@anthropic-ai/claude-code@2.1.113`
  and newer no longer work on Termux. We keep Claude in the launcher
  for users already on the frozen pin; do not upgrade.
- `ollama-termux` targets these Termux package paths:
  - Codex:  `/data/data/com.termux/files/usr/lib/node_modules/@mmmbuto/codex-cli-termux`
  - Qwen:   `/data/data/com.termux/files/usr/lib/node_modules/@mmmbuto/qwen-code-termux`
  - Claude: `/data/data/com.termux/files/usr/lib/node_modules/@anthropic-ai/claude-code`

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
ollama pull qwen3.5:4b
ollama pull gemma4:e4b

# Launch Codex (primary) through Ollama with gemma4:e4b
ollama launch codex --model gemma4:e4b

# Launch Qwen Code (secondary) through Ollama with gemma4:e2b
ollama launch qwen --model gemma4:e2b

# Launch Claude Code (frozen @2.1.112) through Ollama with qwen3.5:4b
ollama launch claude --model qwen3.5:4b
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

An optional `ggml-vulkan` backend is produced when the release is built with
`BUILD_VULKAN=1`. At runtime the fork prepends `/system/lib64` to
`LD_LIBRARY_PATH` so the Android system Vulkan loader is used; that loader
has access to the vendor GPU driver that Termux processes cannot reach
directly (`/vendor/lib64/hw/vulkan.mali.so` on Tensor, `vulkan.adreno.so` on
Qualcomm). Runtime gate: `OLLAMA_VULKAN=1`.

Backend selection is done at runtime by the ggml layer.

---

## Validation Status

Current public release: **`v0.21.0-termux.16`** on npm (`latest`) and GitHub Releases.

Current release target on `develop`: **`v0.21.3-termux.1`**.

On-device validation on Pixel 9 Pro (Tensor G4 / Mali-G715):

- ✅ CPU inference across `armv8.0/8.2/8.6` backends
- ✅ Vulkan GPU offload — 100% layer offload on `gemma4:e2b`, `gemma4:e4b`,
  `medgemma:latest` (see [docs/BENCHMARKS.md](./docs/BENCHMARKS.md))
- ✅ Codex launcher (`@mmmbuto/codex-cli-termux`) via OpenAI-compat `/v1`
- ✅ Qwen Code launcher (`@mmmbuto/qwen-code-termux`) via OpenAI-compat `/v1`
- ✅ Claude Code launcher (`@anthropic-ai/claude-code@2.1.112`, frozen)

Release flow for `v0.21.3-termux.1` and later:

- queue Forge build/package validation on `mgm` from `develop`
- follow that queued build until completion before touching `main`
- verify public-release safety before any public push
- fast-forward clean `main`, then publish GitHub Release and npm package

GitHub release pipeline (`release-termux` + `npm-publish`) produces and publishes:

- `ollama-termux-<version>-android-arm64.tar.gz`
- `ollama-termux-<version>-android-arm64.tar.gz.sha256`
- npm tarball with `install.js` and doc set

---

## License

This project is distributed under the MIT license.

- Original upstream work: [ollama/ollama](https://github.com/ollama/ollama)
- Termux fork work: Davide A. Gugliemi
