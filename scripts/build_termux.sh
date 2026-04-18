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
LINKER_WRAPPER="$BUILD_DIR/aarch64-linux-android28-clang++-filtered"

for tool in "$CLANG" "$CLANGXX" "$LLVM_AR" "$LLVM_RANLIB" "$LLVM_STRIP"; do
    if [ ! -f "$tool" ]; then
        echo "ERROR: Required NDK tool not found at $tool"
        exit 1
    fi
done

echo "=== ollama-termux build (version $VERSION) ==="
echo "NDK: $NDK_ROOT"
echo ""

mkdir -p "$BUILD_DIR"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Go's external linker path for Android can still inherit Linux-only libraries
# such as -lrt and -lpthread. Filter them before invoking the NDK linker.
cat > "$LINKER_WRAPPER" <<EOF
#!/usr/bin/env bash
set -euo pipefail

args=()
for arg in "\$@"; do
    case "\$arg" in
        -lrt|-lpthread) ;;
        *) args+=("\$arg") ;;
    esac
done

exec "$CLANGXX" "\${args[@]}"
EOF
chmod +x "$LINKER_WRAPPER"

# --- Step 1: Build ggml .so backends ---

GGML_VARIANTS=(
    "armv8.0:armv8-a"
    "armv8.2:armv8.2-a+dotprod+fp16"
    "armv8.6:armv8.6-a+dotprod+fp16+i8mm+sve2"
)

# BUILD_VULKAN=1 adds the Vulkan backend alongside the CPU variants.
# Requires glslc in PATH (vulkan-sdk or shaderc on the build host).
# Linked against the NDK Vulkan stub; at runtime the Termux build sets
# LD_LIBRARY_PATH to /system/lib64 so ggml-vulkan resolves the Android
# system loader and reaches the vendor GPU ICD.
BUILD_VULKAN="${BUILD_VULKAN:-0}"
if [ "$BUILD_VULKAN" = "1" ] && ! command -v glslc >/dev/null 2>&1; then
    echo "ERROR: BUILD_VULKAN=1 but glslc not found in PATH"
    echo "       install vulkan-sdk or shaderc on the build host"
    exit 1
fi

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

# --- Step 1b: Optional Vulkan backend ---

if [ "$BUILD_VULKAN" = "1" ]; then
    vulkan_dir="$BUILD_DIR/ggml-vulkan"
    echo "--- Building ggml-vulkan (Android loader, runtime LD_LIBRARY_PATH=/system/lib64) ---"

    # The NDK sysroot ships vulkan/vulkan.h (C) but not vulkan.hpp (C++).
    # ggml-vulkan.cpp needs the C++ wrapper, so point find_package(Vulkan)
    # at the host vulkan-headers package (installed via apt / LunarG SDK).
    VULKAN_HOST_INCLUDE="${VULKAN_HOST_INCLUDE:-/usr/include}"
    if [ ! -f "$VULKAN_HOST_INCLUDE/vulkan/vulkan.hpp" ]; then
        echo "ERROR: vulkan.hpp not found at $VULKAN_HOST_INCLUDE/vulkan/vulkan.hpp"
        echo "       install vulkan-headers (apt install vulkan-headers, or LunarG SDK)"
        exit 1
    fi

    # -isystem in CXX flags injects the host vulkan-headers path ahead of
    # the NDK sysroot so ggml-vulkan.cpp's #include <vulkan/vulkan.hpp>
    # resolves against the C++ wrapper that the NDK sysroot does not ship.
    cmake -S "$ROOT_DIR" -B "$vulkan_dir" \
        -DCMAKE_TOOLCHAIN_FILE="$TOOLCHAIN" \
        -DANDROID_ABI=arm64-v8a \
        -DANDROID_PLATFORM=android-28 \
        -DANDROID_ARM_NEON=ON \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_C_FLAGS="-march=armv8.2-a+dotprod+fp16 -O3" \
        -DCMAKE_CXX_FLAGS="-march=armv8.2-a+dotprod+fp16 -O3 -isystem $VULKAN_HOST_INCLUDE" \
        -DCMAKE_INTERPROCEDURAL_OPTIMIZATION=ON \
        -DGGML_VULKAN=ON \
        -DGGML_VULKAN_CHECK_RESULTS=OFF \
        -DGGML_VULKAN_DEBUG=OFF \
        -DGGML_CUDA=OFF \
        -DGGML_HIP=OFF \
        -DMLX_ENGINE=OFF \
        -DVulkan_INCLUDE_DIR="$VULKAN_HOST_INCLUDE" \
        -GNinja

    ninja -C "$vulkan_dir" ggml-vulkan

    lib_dir="$vulkan_dir/lib/ollama"
    if [ -d "$lib_dir" ]; then
        mkdir -p "$DIST_DIR/lib/ollama/vulkan"
        for so in "$lib_dir"/libggml-vulkan*.so "$lib_dir"/libggml-base*.so; do
            if [ -f "$so" ]; then
                cp "$so" "$DIST_DIR/lib/ollama/vulkan/"
                echo "  Copied: $(basename "$so")"
            fi
        done
    else
        echo "  WARNING: lib/ollama not found in $vulkan_dir"
    fi
    echo ""
fi

# --- Step 2: Cross-compile Go binary ---

echo "--- Building Go binary ---"

export CGO_ENABLED=1
export GOOS=android
export GOARCH=arm64
export CC="$CLANG"
export CXX="$LINKER_WRAPPER"
export LD="$LINKER_WRAPPER"
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
