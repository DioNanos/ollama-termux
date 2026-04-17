#!/usr/bin/env bash
# Cross-compile ollama-termux for Android ARM64 on a Linux host (MGM).
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
VERSION="${VERSION:-0.1.0}"
BUILD_DIR="$ROOT_DIR/build/termux"
DIST_DIR="$ROOT_DIR/dist/termux"

: "${NDK_ROOT:?NDK_ROOT must point to Android NDK installation}"

TOOLCHAIN="$NDK_ROOT/build/cmake/android.toolchain.cmake"
if [ ! -f "$TOOLCHAIN" ]; then
    echo "ERROR: NDK toolchain not found at $TOOLCHAIN"
    exit 1
fi

CLANG="$NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android28-clang"
if [ ! -f "$CLANG" ]; then
    echo "ERROR: NDK clang not found at $CLANG"
    exit 1
fi

echo "=== ollama-termux build (version $VERSION) ==="
echo "NDK: $NDK_ROOT"
echo ""

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
        -GNinja

    ninja -C "$variant_dir" ggml-cpu

    lib_dir="$variant_dir/lib/ollama"
    if [ -d "$lib_dir" ]; then
        mkdir -p "$DIST_DIR/lib/ollama"
        # Map variant name to ggml backend naming convention
        for so in "$lib_dir"/libggml-cpu-*.so; do
            if [ -f "$so" ]; then
                cp "$so" "$DIST_DIR/lib/ollama/"
                echo "  Copied: $(basename "$so")"
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
echo "  scp $TARBALL_PATH pixel9:~/"
echo "  # On Termux:"
echo "  cd /data/data/com.termux/files/usr"
echo "  tar -xzf ~/ollama-termux-$VERSION-android-arm64.tar.gz"
echo "  chmod +x bin/ollama"
