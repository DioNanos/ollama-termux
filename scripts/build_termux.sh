#!/usr/bin/env bash
# Cross-compile ollama-termux for Android ARM64 on a Linux host.
# Produces a tarball with the Go binary + the llama-server runtime\n# (llama-server + ggml CPU variant libraries) under lib/ollama.
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
TERMUX_REUSE_RELEASE_TAG="${TERMUX_REUSE_RELEASE_TAG:-}"
TERMUX_REUSE_RELEASE_REPO="${TERMUX_REUSE_RELEASE_REPO:-DioNanos/ollama-termux}"

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

reuse_prebuilt_termux_libs() {
    local tag="$1"
    local repo="$2"
    local version="${tag#v}"
    local asset="ollama-termux-${version}-android-arm64.tar.gz"
    local url="https://github.com/${repo}/releases/download/${tag}/${asset}"
    local sha_url="${url}.sha256"
    local tmp_dir
    tmp_dir="$(mktemp -d)"
    local archive_path="${tmp_dir}/${asset}"

    echo "--- Reusing prebuilt Termux libs from ${repo}@${tag} ---"
    curl -fsSL "$url" -o "$archive_path"

    if curl -fsSL "$sha_url" -o "${archive_path}.sha256"; then
        (
            cd "$tmp_dir"
            sha256sum -c "$(basename "${archive_path}.sha256")"
        )
    else
        echo "WARNING: no SHA256 file available for ${asset}; continuing without checksum verification"
    fi

    tar -xzf "$archive_path" -C "$tmp_dir"
    if [ ! -d "$tmp_dir/lib/ollama" ]; then
        echo "ERROR: reused release ${tag} did not contain lib/ollama"
        exit 1
    fi

    mkdir -p "$DIST_DIR/lib/ollama"
    cp -R "$tmp_dir/lib/ollama/." "$DIST_DIR/lib/ollama/"
    rm -rf "$tmp_dir"
    echo ""
}

# --- Step 1: Build or reuse the llama-server runtime (lib/ollama) ---
#
# Upstream removed the in-repo CGO/ggml engines (#16031): all GGML models run
# via the llama-server subprocess built from the pinned llama.cpp source
# (LLAMA_CPP_VERSION) through llama/server/CMakeLists.txt. FetchContent needs
# network access on first configure.
#
# GGML_BACKEND_DL=ON + GGML_CPU_ALL_VARIANTS=ON produce the runtime-dispatched
# CPU variant libraries (libggml-cpu-*.so) next to llama-server, mirroring the
# upstream `cpu` preset. Set TERMUX_CPU_VARIANTS=OFF to fall back to a single
# armv8.2 tuned build if the variant matrix fails for the Android toolchain.
#
# BUILD_VULKAN=1 enables the llama.cpp Vulkan backend (needs glslc on the
# host). At runtime the Go side (llm/termux.go) exposes /system/lib64 so
# dlopen can reach the Android system Vulkan loader and the vendor GPU ICD.

BUILD_VULKAN="${BUILD_VULKAN:-0}"
TERMUX_CPU_VARIANTS="${TERMUX_CPU_VARIANTS:-ON}"

if [ -n "$TERMUX_REUSE_RELEASE_TAG" ]; then
    reuse_prebuilt_termux_libs "$TERMUX_REUSE_RELEASE_TAG" "$TERMUX_REUSE_RELEASE_REPO"
else
    if [ "$BUILD_VULKAN" = "1" ] && ! command -v glslc >/dev/null 2>&1; then
        echo "ERROR: BUILD_VULKAN=1 but glslc not found in PATH"
        echo "       install vulkan-sdk or shaderc on the build host"
        exit 1
    fi

    # Vulkan cross-build hints: find_package(Vulkan) cannot resolve under the
    # NDK toolchain on its own. The NDK sysroot ships vulkan/*.h and
    # libvulkan.so but not the C++ wrapper (vulkan.hpp), which ggml-vulkan
    # needs, so build an include overlay: NDK C headers + host-only .hpp.
    # Never add /usr/include itself to the Android include path; that leaks
    # glibc headers into the NDK sysroot.
    if [ "$BUILD_VULKAN" = "1" ]; then
        VULKAN_HOST_INCLUDE="${VULKAN_HOST_INCLUDE:-/usr/include}"
        VULKAN_NDK_INCLUDE="$NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64/sysroot/usr/include"
        VULKAN_NDK_LIBRARY="$NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64/sysroot/usr/lib/aarch64-linux-android/28/libvulkan.so"
        VULKAN_INCLUDE_OVERLAY="$BUILD_DIR/vulkan-include-overlay"

        if [ ! -f "$VULKAN_HOST_INCLUDE/vulkan/vulkan.hpp" ]; then
            echo "ERROR: vulkan.hpp not found at $VULKAN_HOST_INCLUDE/vulkan/vulkan.hpp"
            echo "       install libvulkan-dev or LunarG SDK on the build host"
            exit 1
        fi
        if [ ! -f "$VULKAN_NDK_INCLUDE/vulkan/vulkan.h" ] || [ ! -f "$VULKAN_NDK_LIBRARY" ]; then
            echo "ERROR: NDK Vulkan headers/library not found under $NDK_ROOT"
            exit 1
        fi

        rm -rf "$VULKAN_INCLUDE_OVERLAY"
        mkdir -p "$VULKAN_INCLUDE_OVERLAY/vulkan"
        cp "$VULKAN_NDK_INCLUDE"/vulkan/*.h "$VULKAN_INCLUDE_OVERLAY/vulkan/"
        if [ -d "$VULKAN_NDK_INCLUDE/vk_video" ]; then
            cp -R "$VULKAN_NDK_INCLUDE/vk_video" "$VULKAN_INCLUDE_OVERLAY/"
        fi
        cp "$VULKAN_HOST_INCLUDE"/vulkan/*.hpp "$VULKAN_INCLUDE_OVERLAY/vulkan/"

        # ggml-vulkan.cpp includes <spirv/unified1/spirv.hpp>. The host
        # /usr/include is an implicit dir CMake filters out under the NDK
        # toolchain, so stage the SPIRV headers inside the overlay too.
        SPIRV_HOST_INCLUDE="${SPIRV_HOST_INCLUDE:-/usr/include/spirv}"
        if [ ! -f "$SPIRV_HOST_INCLUDE/unified1/spirv.hpp" ]; then
            echo "ERROR: spirv/unified1/spirv.hpp not found at $SPIRV_HOST_INCLUDE"
            echo "       install spirv-headers (apt) or set SPIRV_HOST_INCLUDE"
            exit 1
        fi
        cp -R "$SPIRV_HOST_INCLUDE" "$VULKAN_INCLUDE_OVERLAY/"

        # ggml-vulkan also needs the host SPIRV-Headers cmake package; the
        # Android toolchain re-roots find_package, so locate the config on
        # the host and pass SPIRV-Headers_DIR explicitly.
        SPIRV_HEADERS_DIR="${SPIRV_HEADERS_DIR:-}"
        if [ -z "$SPIRV_HEADERS_DIR" ]; then
            for d in /usr/lib/cmake/SPIRV-Headers /usr/share/cmake/SPIRV-Headers /usr/lib/x86_64-linux-gnu/cmake/SPIRV-Headers; do
                if [ -f "$d/SPIRV-HeadersConfig.cmake" ]; then
                    SPIRV_HEADERS_DIR="$d"
                    break
                fi
            done
        fi
        if [ -z "$SPIRV_HEADERS_DIR" ]; then
            echo "ERROR: SPIRV-Headers cmake package not found on the host"
            echo "       install spirv-headers (apt) or set SPIRV_HEADERS_DIR"
            exit 1
        fi
    fi

    server_dir="$BUILD_DIR/llama-server"
    echo "--- Building llama-server (llama.cpp $(cat "$ROOT_DIR/LLAMA_CPP_VERSION")) ---"

    cmake_args=(
        -S "$ROOT_DIR/llama/server" -B "$server_dir"
        -DCMAKE_TOOLCHAIN_FILE="$TOOLCHAIN"
        -DANDROID_ABI=arm64-v8a
        -DANDROID_PLATFORM=android-28
        -DANDROID_ARM_NEON=ON
        -DCMAKE_BUILD_TYPE=Release
        -DBUILD_SHARED_LIBS=ON
        -DGGML_BACKEND_DL=ON
        -DGGML_NATIVE=OFF
        -DGGML_OPENMP=OFF
        -DGGML_CPU_ALL_VARIANTS="$TERMUX_CPU_VARIANTS"
        -DOLLAMA_RUNNER_DIR=""
        -DCMAKE_INSTALL_PREFIX="$DIST_DIR"
        -GNinja
    )
    if [ "$TERMUX_CPU_VARIANTS" != "ON" ]; then
        cmake_args+=(
            -DCMAKE_C_FLAGS="-march=armv8.2-a+dotprod+fp16 -O3"
            -DCMAKE_CXX_FLAGS="-march=armv8.2-a+dotprod+fp16 -O3"
        )
    fi
    cmake "${cmake_args[@]}"
    cmake --build "$server_dir" -- -l "$(nproc)"
    cmake --install "$server_dir" --component llama-server --strip

    if [ ! -x "$DIST_DIR/lib/ollama/llama-server" ]; then
        echo "ERROR: llama-server missing from $DIST_DIR/lib/ollama after install"
        exit 1
    fi
    echo "  Installed runtime:"
    ls -1 "$DIST_DIR/lib/ollama" | sed 's/^/    /'
    echo ""

    # --- Step 1b: Optional Vulkan backend (separate pass, upstream model) ---
    #
    # Mirrors the upstream `vulkan` preset: a dedicated configure with
    # OLLAMA_RUNNER_DIR=vulkan + OLLAMA_GPU_BACKEND=vulkan installs ONLY the
    # libggml-vulkan backend into lib/ollama/vulkan/, which the runner
    # discovers at runtime (GGML_BACKEND_DL).
    if [ "$BUILD_VULKAN" = "1" ]; then
        vulkan_dir="$BUILD_DIR/llama-server-vulkan"
        echo "--- Building Vulkan backend (llama.cpp $(cat "$ROOT_DIR/LLAMA_CPP_VERSION")) ---"

        cmake -S "$ROOT_DIR/llama/server" -B "$vulkan_dir" \
            -DCMAKE_TOOLCHAIN_FILE="$TOOLCHAIN" \
            -DANDROID_ABI=arm64-v8a \
            -DANDROID_PLATFORM=android-28 \
            -DANDROID_ARM_NEON=ON \
            -DCMAKE_BUILD_TYPE=Release \
            -DBUILD_SHARED_LIBS=ON \
            -DGGML_BACKEND_DL=ON \
            -DGGML_NATIVE=OFF \
            -DGGML_OPENMP=OFF \
            -DGGML_VULKAN=ON \
            -DOLLAMA_RUNNER_DIR=vulkan \
            -DOLLAMA_GPU_BACKEND=vulkan \
            -DVulkan_INCLUDE_DIR="$VULKAN_INCLUDE_OVERLAY" \
            -DVulkan_LIBRARY="$VULKAN_NDK_LIBRARY" \
            -DVulkan_GLSLC_EXECUTABLE="$(command -v glslc)" \
            -DSPIRV-Headers_DIR="$SPIRV_HEADERS_DIR" \
            -DCMAKE_INSTALL_PREFIX="$DIST_DIR" \
            -GNinja
        cmake --build "$vulkan_dir" -- -l "$(nproc)"
        cmake --install "$vulkan_dir" --component llama-server --strip

        if ! ls "$DIST_DIR/lib/ollama/vulkan/"libggml-vulkan*.so >/dev/null 2>&1; then
            echo "ERROR: libggml-vulkan missing from $DIST_DIR/lib/ollama/vulkan after install"
            exit 1
        fi
        echo "  Installed Vulkan backend:"
        ls -1 "$DIST_DIR/lib/ollama/vulkan" | sed 's/^/    /'
        echo ""
    fi
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
    # Whole runtime tree: llama-server + libllama/libggml-base + CPU variant
    # libraries (+ optional GPU backend subdirectories).
    cp -R "$DIST_DIR/lib/ollama/." "$STAGING/lib/ollama/"
    chmod +x "$STAGING/lib/ollama/llama-server" 2>/dev/null || true
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
