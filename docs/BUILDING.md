# Building ollama-termux

## Overview

There are two ways to install:

1. **Pre-built (recommended)** — Download release tarball via `npm install -g @mmmbuto/ollama-termux`
2. **Cross-compile from MGM** — Build Go binary + ggml `.so` on a Linux desktop, deploy to phone

## 1. Pre-built Install (Termux)

```bash
npm install -g @mmmbuto/ollama-termux
```

This downloads a tarball with the Go binary and all ggml backends. No toolchain needed.

## 2. Cross-Compile from MGM (Full Build)

Use `scripts/build_termux.sh` for the canonical build:

```bash
export NDK_ROOT=~/android-ndk/android-ndk-r27c
./scripts/build_termux.sh
```

Output: `dist/ollama-termux-<version>-android-arm64.tar.gz`

### Prerequisites on MGM

```bash
# Android NDK r27c (LLVM 18, supports ARMv8.2+ targets)
mkdir -p ~/android-ndk
cd ~/android-ndk
wget https://dl.google.com/android/repository/android-ndk-r27c-linux.zip
unzip android-ndk-r27c-linux.zip
export NDK_ROOT=~/android-ndk/android-ndk-r27c

# Go >= 1.24, CMake >= 3.22, Ninja
```

### CMake Cross-Compilation Patches

The following patches are needed for cross-compilation with NDK:

**mem_hip.cpp** - Add glob.h include:
```diff
+#include <glob.h>
```

**CMakeLists.txt** - Wrap install() in CMAKE_CROSSCOMPILING check:
```cmake
if(NOT CMAKE_CROSSCOMPILING)
install(TARGETS ggml-base ${CPU_VARIANTS}
    ...
)
endif()
```

### Deploy to Phone

```bash
# Via SSH (sshd running on Termux, port 8022)
scp dist/ollama-termux-*-android-arm64.tar.gz pixel9:~/

# On Termux:
cd /data/data/com.termux/files/usr
tar -xzf ~/ollama-termux-*-android-arm64.tar.gz
chmod +x bin/ollama
```

### Verify Backend Selection

```bash
# On Termux, check which backend is loaded
ollama serve --verbose 2>&1 | grep -i "ggml\|cpu\|backend"
```

The ARM feature detection (HWCAP) in ggml automatically selects the best `.so` variant.

## 3. Manual ggml Build (One Variant)

If you only need one variant:

```bash
mkdir -p build/termux && cd build/termux
cmake ../.. \
  -DCMAKE_TOOLCHAIN_FILE=$NDK_ROOT/build/cmake/android.toolchain.cmake \
  -DANDROID_ABI=arm64-v8a \
  -DANDROID_PLATFORM=android-28 \
  -DANDROID_ARM_NEON=ON \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_INTERPROCEDURAL_OPTIMIZATION=ON \
  -DCMAKE_C_FLAGS="-march=armv8.2-a+dotprod+fp16 -O3" \
  -DCMAKE_CXX_FLAGS="-march=armv8.2-a+dotprod+fp16 -O3" \
  -DGGML_VULKAN=OFF \
  -DGGML_CUDA=OFF \
  -GNinja
ninja ggml-cpu
```

## 4. Dependencies Checklist

### On Termux
- [x] `nodejs-lts` >= 18
- [x] `@anthropic-ai/claude-code` (npm)
- [x] `@mmmbuto/codex-cli-termux` (npm)
- [x] `golang` >= 1.24 (only for manual on-device build)

### On MGM
- [x] Android NDK r27c+
- [x] Go >= 1.24
- [x] CMake >= 3.22
- [x] Ninja
- [ ] ~10 GB free disk space

## 5. Build Targets

| Target | Architecture | Flags | Device |
|--------|-------------|-------|--------|
| armv8.0 | ARMv8.0-A | baseline | Old phones (fallback) |
| armv8.2 | ARMv8.2-A +dotprod +fp16 | `-march=armv8.2-a+dotprod+fp16` | Pixel 9 Pro, Galaxy S24+, Galaxy S25 |
| armv8.6 | ARMv8.6-A +dotprod +fp16 +i8mm +sve2 | `-march=armv8.6-a+dotprod+fp16+i8mm+sve2` | Pixel 9 Pro (Cortex-X4), Galaxy S25 Ultra (Oryon) |

Note: `i8mm` is an ARMv8.6-A feature. The armv8.2 build deliberately omits it for broader compatibility. The armv8.6 build targets newer cores (Cortex-X4, Oryon) that support the full feature set including SVE2.
