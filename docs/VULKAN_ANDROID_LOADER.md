# Vulkan on Termux via Android System Loader

## Problem

Vanilla Termux cannot reach the vendor GPU driver. `pkg install vulkan-tools`
installs a Mesa-based loader that only discovers `llvmpipe` (CPU software).
The real vendor ICD lives at `/vendor/lib64/hw/vulkan.*.so` and is blocked
for `dlopen()` by the Android linker namespace Termux processes run under.
The loader permitted_paths list includes `/system/lib64/hw` but not
`/vendor/lib64/hw`:

```
permitted_paths=
  /system/lib64/drm:/system/lib64/extractors:/system/lib64/hw:
  /system_ext/lib64:...:/system/lib64/bootstrap
```

## Solution

Use the Android system Vulkan loader at `/system/lib64/libvulkan.so`. That
loader runs in the root namespace which has access to `/vendor/lib64/hw/`,
so when it scans for ICDs it finds the real driver. No root, no chroot, no
`VK_ICD_FILENAMES` manifest.

The only trick is path ordering: Termux binaries default to searching
`/data/data/com.termux/files/usr/lib` first, which contains the Termux
Vulkan loader. Prepending `/system/lib64` makes the Android loader win.

## Implementation

`llm/server.go:StartRunner` prepends `/system/lib64` to `LD_LIBRARY_PATH`
when `isTermux()` is true. The runner subprocess inherits that, so when
ggml-vulkan calls `dlopen("libvulkan.so")` it resolves the system loader.

Build side: `scripts/build_termux.sh` adds an optional `BUILD_VULKAN=1`
flag that compiles `ggml-vulkan` alongside the CPU backends. Shaders are
compiled at build time with `glslc` on the host.

## Verification

Tested on Pixel 9 Pro (Tensor G4, Mali-G715 MC7) with `scripts/probe/vulkan_probe.c`:

```
$ LD_LIBRARY_PATH=/system/lib64 /tmp/vkprobe
physical devices: 1
  [0] name=Mali-G715
       type=INTEGRATED_GPU  api=1.4.305  driver=0xd802000
       vendor=0x13b5 device=0xb8a20000
```

Without the override the same binary sees `llvmpipe` only. See
`scripts/probe/README.md` for the one-liner build.

## Known limitations

- Only tested on Pixel 9 Pro. Other SoCs (Snapdragon with Adreno, MediaTek
  with Mali) should work identically provided `/vendor/lib64/hw/vulkan.*.so`
  exists and the OEM exposes Vulkan 1.1+.
- The Android system loader version varies per device. Feature support
  (coopmat, shaderFloat16, shaderInt8, subgroup ops) must be probed at
  runtime and the shader pipeline configured accordingly — ggml-vulkan
  already handles this.
- Integrated GPUs share system RAM. KV cache on GPU competes with model
  weights for the same pool; prefer host-visible coherent allocations
  and be conservative with `-ngl`.
- OLLAMA_VULKAN=1 is still required to opt in at runtime (upstream gate).

## Vendor driver map

| SoC | Device | vendorID | Driver path |
|---|---|---|---|
| Tensor G4 | Pixel 9/9 Pro | 0x13b5 (ARM) | `/vendor/lib64/hw/vulkan.mali.so` |
| Tensor G5 | Pixel 10/10 Pro | 0x13b5 (ARM) | `/vendor/lib64/hw/vulkan.mali.so` |
| Snapdragon 8 Gen 3 | Galaxy S24+ | 0x5143 (Qualcomm) | `/vendor/lib64/hw/vulkan.adreno.so` |
| Snapdragon 8 Elite | Galaxy S25U | 0x5143 (Qualcomm) | `/vendor/lib64/hw/vulkan.adreno.so` |
