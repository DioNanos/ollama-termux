#!/usr/bin/env bash
# Cross-compile ollama-termux for Android ARM64 on a Linux host.
# Produces a tarball with Go binary + optimized ggml .so backends.
#
# Prerequisites:
#   - Android NDK r27c+ (set NDK_ROOT)
#   - Go >= 1.24 with cross-compilation support
#   - CMake >= 3.22, Ninja
#
# Usage:
#   export NDK_ROOT=~/android-ndk/android-ndk-r27c
#   ./scripts/build_termux.sh
#
# Output: dist/termux/ollama-termux-<version>-android-arm64.tar.gz

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
if [ -z "${VERSION:-}" ]; then
    if command -v node >/dev/null 2>&1; then
        VERSION="$(node -p "require('$ROOT_DIR/package.json').version")"
    else
        VERSION="$(sed -n 's/.*"version":[[:space:]]*"\([^"]*\)".*/\1/p' "$ROOT_DIR/package.json" | head -1)"
    fi
fi
BUILD_DIR="$ROOT_DIR/build/termux"
DIST_DIR="$ROOT_DIR/dist/termux"

: "${NDK_ROOT:?NDK_ROOT must point to Android NDK installation}"

TOOLCHAIN="$NDK_ROOT/build/cmake/android.toolchain.cmake"
if [ ! -f "$TOOLCHAIN" ]; then
    echo "ERROR: NDK toolchain not found at $TOOLCHAIN"
    exit 1
fi

TOOLBIN="$NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64/bin"
CLANG="$TOOLBIN/aarch64-linux-android28-clang"
CLANGXX="$TOOLBIN/aarch64-linux-android28-clang++"
LLVM_AR="$TOOLBIN/llvm-ar"
LLVM_RANLIB="$TOOLBIN/llvm-ranlib"
LLVM_STRIP="$TOOLBIN/llvm-strip"

for tool in "$CLANG" "$CLANGXX" "$LLVM_AR" "$LLVM_RANLIB" "$LLVM_STRIP"; do
    if [ ! -f "$tool" ]; then
        echo "ERROR: Required NDK tool not found at $tool"
        exit 1
    fi
done

echo "=== ollama-termux build (version $VERSION) ==="
echo "NDK: $NDK_ROOT"
echo ""

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# --- Step 1: Build ggml .so backends ---

GGML_VARIANTS=(
    "armv8.0:armv8-a"
    "armv8.2:armv8.2-a+dotprod+fp16"
    "armv8.6:armv8.6-a+dotprod+fp16+i8mm+sve2"
)

for variant in "${GGML_VARIANTS[@]}"; do
    IFS=':' read -r name march <<< "$variant"
    variant_dir="$BUILD_DIR/ggml-$name"
    echo "--- Building ggml variant: $name ($march) ---"

    cmake -S "$ROOT_DIR" -B "$variant_dir" \
        -DCMAKE_TOOLCHAIN_FILE="$TOOLCHAIN" \
        -DANDROID_ABI=arm64-v8a \
        -DANDROID_PLATFORM=android-28 \
        -DANDROID_ARM_NEON=ON \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_C_FLAGS="-march=$march -O3" \
        -DCMAKE_CXX_FLAGS="-march=$march -O3" \
        -DCMAKE_INTERPROCEDURAL_OPTIMIZATION=ON \
        -DGGML_VULKAN=OFF \
        -DGGML_CUDA=OFF \
        -DGGML_HIP=OFF \
        -DMLX_ENGINE=OFF \
        -DCMAKE_DISABLE_FIND_PACKAGE_Vulkan=TRUE \
        -GNinja

    ninja -C "$variant_dir" ggml-cpu

    lib_dir="$variant_dir/lib/ollama"
    if [ -d "$lib_dir" ]; then
        mkdir -p "$DIST_DIR/lib/ollama"
        # Rename each .so with the variant name so the 3 builds don't overwrite each other.
        # Output: libggml-cpu-android_armv8_0_1.so / *_armv8_2_1.so / *_armv8_6_1.so
        suffix="android_${name//./_}_1"
        for so in "$lib_dir"/libggml-cpu*.so; do
            if [ -f "$so" ]; then
                base="$(basename "$so" .so)"
                dest="$DIST_DIR/lib/ollama/${base}-${suffix}.so"
                cp "$so" "$dest"
                echo "  Copied: $(basename "$dest")"
            fi
        done
    else
        echo "  WARNING: lib/ollama not found in $variant_dir"
    fi
    echo ""
done

# --- Step 2: Cross-compile Go binary ---

echo "--- Building Go binary ---"

export CGO_ENABLED=1
export GOOS=android
export GOARCH=arm64
export CC="$CLANG"
export CXX="$CLANGXX"
export LD="$CLANGXX"
export AR="$LLVM_AR"
export RANLIB="$LLVM_RANLIB"
export STRIP="$LLVM_STRIP"
export CGO_CFLAGS="-O3"
export CGO_CXXFLAGS="-O3"
export CGO_LDFLAGS="-llog"

mkdir -p "$DIST_DIR/bin"

go build \
    -o "$DIST_DIR/bin/ollama" \
    -ldflags="-s -w -X github.com/ollama/ollama/version.Version=$VERSION" \
    -trimpath \
    "$ROOT_DIR"

echo "  Built: $DIST_DIR/bin/ollama"
file "$DIST_DIR/bin/ollama"
echo ""

# --- Step 3: Package tarball ---

echo "--- Packaging ---"

TARBALL_NAME="ollama-termux-$VERSION-android-arm64.tar.gz"
TARBALL_PATH="$ROOT_DIR/dist/$TARBALL_NAME"

mkdir -p "$(dirname "$TARBALL_PATH")"
rm -f "$TARBALL_PATH" "$TARBALL_PATH.sha256"

# Create temp staging dir with clean structure
STAGING=$(mktemp -d)
mkdir -p "$STAGING/bin" "$STAGING/lib/ollama"

cp "$DIST_DIR/bin/ollama" "$STAGING/bin/"
if [ -d "$DIST_DIR/lib/ollama" ]; then
    cp "$DIST_DIR/lib/ollama/"*.so "$STAGING/lib/ollama/" 2>/dev/null || true
fi

# Add install helper
cp "$ROOT_DIR/install.js" "$STAGING/install.js"

tar -czf "$TARBALL_PATH" -C "$STAGING" .
rm -rf "$STAGING"

echo "  Package: $TARBALL_PATH"
echo "  Size: $(du -h "$TARBALL_PATH" | cut -f1)"
echo ""

# --- Step 4: SHA256 ---

cd "$(dirname "$TARBALL_PATH")"
sha256sum "$(basename "$TARBALL_PATH")" > "$(basename "$TARBALL_PATH")".sha256
echo "  SHA256: $(cat "$(basename "$TARBALL_PATH")".sha256)"
echo ""

echo "=== Build complete ==="
echo "Deploy to Termux:"
echo "  scp $TARBALL_PATH <device>:~/"
echo "  # On Termux:"
echo "  cd /data/data/com.termux/files/usr"
echo "  tar -xzf ~/ollama-termux-$VERSION-android-arm64.tar.gz"
echo "  chmod +x bin/ollama"
