# Building ollama-termux

## Overview

There are two components to build:

1. **Go binary** — can be built directly on Termux
2. **ggml shared libraries (.so)** — must be cross-compiled from a Linux desktop (MGM)

## 1. Go Binary (Termux Native)

```bash
pkg install golang -y
git clone https://github.com/DioNanos/ollama-termux.git
cd ollama-termux
go build -o ollama-termux -ldflags="-s -w" -trimpath .
```

## 2. ggml Libraries (Cross-Compile on MGM)

### Prerequisites on MGM (Linux Mint, x86_64)

```bash
# Docker (if not installed)
sudo apt update && sudo apt install docker.io -y
sudo usermod -aG docker $USER
newgrp docker

# Android NDK r27c (LLVM 18, supports ARMv8.2+ targets)
mkdir -p ~/android-ndk
cd ~/android-ndk
wget https://dl.google.com/android/repository/android-ndk-r27c-linux.zip
unzip android-ndk-r27c-linux.zip
export NDK_ROOT=~/android-ndk/android-ndk-r27c
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

**CMakeLists.txt** - Fix Vulkan find_package:
```cmake
find_package(Vulkan COMPONENTS glslc)
```

### Build Command

```bash
cd ~/Dev/60_toolchains/ollama-termux
mkdir -p build/termux && cd build/termux
cmake ../.. \
  -DCMAKE_TOOLCHAIN_FILE=$NDK_ROOT/build/cmake/android.toolchain.cmake \
  -DANDROID_ABI=arm64-v8a \
  -DANDROID_PLATFORM=android-28 \
  -DANDROID_ARM_NEON=ON \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_C_FLAGS="-march=armv8.2-a+dotprod+i8mm+fp16 -O3 -flto" \
  -DCMAKE_CXX_FLAGS="-march=armv8.2-a+dotprod+i8mm+fp16 -O3 -flto" \
  -DGGML_VULKAN=OFF \
  -DGGML_CUDA=OFF \
  -GNinja
ninja ggml-cpu
```

### Copy .so to Phone

```bash
# Files are at:
# build/termux/lib/ollama/libggml-cpu-android_armv8.2_1.so

# Copy via SSH to Pixel 9 Pro (Termux)
scp -P 8022 build/termux/lib/ollama/libggml-cpu-android_armv8.2_1.so dag@localhost:/data/data/com.termux/files/usr/lib/ollama/
```

### Verify Backend Selection

```bash
# On Termux, check which backend is loaded
ollama serve --verbose 2>&1 | grep -i "ggml\|cpu\|backend"
```

The ARM feature detection (HWCAP) in ggml automatically selects the best `.so` variant.

## 3. Dependencies Checklist

### On Termux
- [x] `golang` >= 1.24
- [x] `nodejs-lts` >= 18
- [x] `@anthropic-ai/claude-code` (npm)
- [x] `@mmmbuto/codex-cli-termux` (npm)
- [x] Existing ollama package (`pkg install ollama`) for ggml `.so` files

### On MGM
- [x] Docker
- [x] Android NDK r27c+
- [x] CMake >= 3.22
- [x] GCC/Clang with ARM cross-compilation support
- [ ] ~10 GB free disk space

## 4. Build Targets

| Target | Architecture | Flags | Device |
|--------|-------------|-------|--------|
| armv8.0 | ARMv8.0-A | baseline | Old phones (fallback) |
| armv8.2 | ARMv8.2-A +dotprod +fp16 | `-march=armv8.2-a+dotprod+fp16` | Pixel 9 Pro, Galaxy S24+ |
| armv8.6 | ARMv8.6-A +sve +i8mm | `-march=armv8.6-a+dotprod+fp16+i8mm+sve` | Future devices |

For Pixel 9 Pro specifically: `armv8.2` with `dotprod` + `fp16` provides the best balance of performance and compatibility.
