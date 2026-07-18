# Building ollama-termux

## Overview

`ollama-termux` is published through two linked artifacts for the same version:

- GitHub Release assets:
  - `ollama-termux-<version>-android-arm64.tar.gz`
  - `ollama-termux-<version>-android-arm64.tar.gz.sha256`
- npm package:
  - `@mmmbuto/ollama-termux`

The npm installer is only valid when the matching GitHub Release assets already
exist.

## Versioning

The fork tracks upstream Ollama versions and appends a Termux release suffix:

- upstream base: `0.22.1`
- fork release: `0.22.1-termux.1`
- git tag: `v0.22.1-termux.1`

`package.json` is the source of truth for the fork version. The build script and
release workflow read from it.

## Local Cross-Build

Canonical local build command:

```bash
export NDK_ROOT=~/android-ndk/android-ndk-r27c
./scripts/build_termux.sh
```

Outputs:

- `dist/ollama-termux-<version>-android-arm64.tar.gz`
- `dist/ollama-termux-<version>-android-arm64.tar.gz.sha256`

## Prerequisites

On the Linux build host:

- Android NDK `r27c`
- the Go version declared by upstream in `go.mod`
- Node.js (used to read package version)
- Python 3 (release archive validation)
- CMake `>= 3.24`
- Ninja
- `file`, `curl`, `unzip`
- For `BUILD_VULKAN=1`: `glslc`, `libvulkan-dev` (vulkan.hpp), `spirv-headers`
- Network access on first configure (llama.cpp is fetched at the pinned
  `LLAMA_CPP_VERSION` via CMake FetchContent)

Example NDK setup:

```bash
mkdir -p ~/android-ndk
cd ~/android-ndk
wget https://dl.google.com/android/repository/android-ndk-r27c-linux.zip
unzip android-ndk-r27c-linux.zip
export NDK_ROOT=~/android-ndk/android-ndk-r27c
```

## Build Artifacts

Since the 0.30.x line, inference runs through the upstream `llama-server`
subprocess; the tarball ships the whole runtime:

- `bin/ollama`
- `lib/ollama/llama-server` (plus `llama-quantize`)
- `lib/ollama/libllama*.so`, `lib/ollama/libggml-base.so`, `lib/ollama/libmtmd.so`
- `lib/ollama/libggml-cpu-android_armv*.so` — runtime-dispatched CPU variants
  (armv8.0 through armv9.2, selected per device at startup)
- `lib/ollama/vulkan/libggml-vulkan.so` when built with `BUILD_VULKAN=1`

The installer downloads that tarball and extracts it into the Termux prefix.
Set `TERMUX_CPU_VARIANTS=OFF` to build a single armv8.2-tuned CPU backend
instead of the full variant matrix.

## Release Workflow

1. Update `package.json` version to the next `x.y.z-termux.N`
2. Run `release-termux` manually with `publish_release=0`, the candidate branch
   as `source_ref`, and `build_vulkan=1`
3. Audit the uploaded tarball and checksum, then merge the sanitized candidate
   to GitHub `main`
4. Create the exact annotated tag `v<version>` only after approval
5. The tag workflow rebuilds from that tag, validates its archive and publishes
   the GitHub Release
6. `.github/workflows/npm-publish.yaml` downloads and re-verifies the exact
   release assets, rejects npm version collisions, and publishes stable builds
   explicitly to `latest`

Published releases are fail-closed: the source must be the exact tag, Android
NDK r27c is size/checksum-bound, Vulkan is required, and the npm installer will
not install an asset without a matching valid SHA-256 file.

## Manual Device Install

```bash
# Copy tarball to your Android device
scp dist/ollama-termux-*-android-arm64.tar.gz <device>:~/

# On Termux
cd /data/data/com.termux/files/usr
tar -xzf ~/ollama-termux-*-android-arm64.tar.gz
chmod +x bin/ollama
```

## Runtime Notes

- Thread selection uses big-core detection via `cpufreq` when available
- Free memory starts from Linux `MemAvailable`, is never inflated, reserves at
  least 1 GiB (or 12.5% on larger devices) for Android, and is reduced again
  when less than 10% of zram/swap remains
- Flash attention follows llama-server `--flash-attn auto`
- The best CPU backend variant is chosen at runtime by ggml feature detection
- `/system/lib64` precedes `$PREFIX/lib` in the `llama-server` subprocess
  library path so the Android vendor Vulkan loader cannot be shadowed by Mesa
