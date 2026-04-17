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

- upstream base: `0.21.0`
- fork release: `0.21.0-termux.1`
- git tag: `v0.21.0-termux.1`

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
- Go `>= 1.24`
- Node.js (used to read package version)
- CMake `>= 3.22`
- Ninja
- `file`, `curl`, `unzip`

Example NDK setup:

```bash
mkdir -p ~/android-ndk
cd ~/android-ndk
wget https://dl.google.com/android/repository/android-ndk-r27c-linux.zip
unzip android-ndk-r27c-linux.zip
export NDK_ROOT=~/android-ndk/android-ndk-r27c
```

## Build Artifacts

The tarball contains:

- `bin/ollama`
- `lib/ollama/libggml-cpu-android_armv8_0_1.so`
- `lib/ollama/libggml-cpu-android_armv8_2_1.so`
- `lib/ollama/libggml-cpu-android_armv8_6_1.so`

The installer downloads that tarball and extracts it into the Termux prefix.

## Release Workflow

1. Update `package.json` version to the next `x.y.z-termux.N`
2. Push the corresponding tag `v<version>`
3. GitHub Actions runs `.github/workflows/release.yaml`
4. The workflow builds the Android tarball and checksum
5. The workflow publishes the GitHub Release and uploads both assets
6. `.github/workflows/npm-publish.yaml` verifies those assets exist
7. npm publish can proceed safely

## Manual Device Install

```bash
# Copy tarball to the phone
scp dist/ollama-termux-*-android-arm64.tar.gz pixel9:~/

# On Termux
cd /data/data/com.termux/files/usr
tar -xzf ~/ollama-termux-*-android-arm64.tar.gz
chmod +x bin/ollama
```

## Runtime Notes

- Thread selection uses big-core detection via `cpufreq` when available
- Free memory is derived from an Android-specific `MemTotal` heuristic
- Flash attention is enabled automatically for the tuned mobile path
- The highest CPU backend is chosen at runtime by ggml feature detection
