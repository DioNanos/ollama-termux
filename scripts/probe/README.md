# Vulkan Probe

Micro-utility that proves the Android vendor GPU driver is reachable from a
Termux-launched process when the Android system Vulkan loader is preferred
over the Termux loader.

## Why

On Termux, `pkg install vulkan-tools` provides a Mesa-based loader that scans
`/usr/share/vulkan/icd.d` and only finds `llvmpipe` (CPU software rasterizer).
The real vendor driver (`/vendor/lib64/hw/vulkan.mali.so` on Tensor, or
`vulkan.adreno.so` on Qualcomm) is blocked for `dlopen()` by the Android
linker namespace under which Termux runs.

Workaround: link against the Android system loader at `/system/lib64/libvulkan.so`.
That loader runs under a namespace that has access to the vendor HAL, so
`vkEnumeratePhysicalDevices()` returns the real GPU.

## Build

```
clang scripts/probe/vulkan_probe.c \
  -I $PREFIX/include \
  -L /system/lib64 -lvulkan \
  -o /tmp/vkprobe
```

## Run

The binary RUNPATH normally still favors the Termux loader, so override
explicitly at runtime:

```
LD_LIBRARY_PATH=/system/lib64 /tmp/vkprobe
```

Expected on Pixel 9 Pro:

```
physical devices: 1
  [0] name=Mali-G715
       type=INTEGRATED_GPU  api=1.4.305  driver=0xd802000  vendor=0x13b5 device=0xb8a20000
```

Expected without the override (still Termux loader):

```
physical devices: 1
  [0] name=llvmpipe (LLVM 21.1.8, 128 bits)
       type=CPU  api=1.4.335  driver=0x6800004  vendor=0x10005 device=0x0
```

## Vendor mapping

| SoC | vendorID | driver path |
|---|---|---|
| Tensor G4 (Pixel 9 Pro) | 0x13b5 (ARM) | `/vendor/lib64/hw/vulkan.mali.so` |
| Tensor G5 (Pixel 10 Pro) | 0x13b5 (ARM) | `/vendor/lib64/hw/vulkan.mali.so` |
| Snapdragon 8 Gen 3 (S24+) | 0x5143 (Qualcomm) | `/vendor/lib64/hw/vulkan.adreno.so` |
| Snapdragon 8 Elite (S25U) | 0x5143 (Qualcomm) | `/vendor/lib64/hw/vulkan.adreno.so` |

## Why this matters for ollama-termux

When the runner subprocess is spawned with `LD_LIBRARY_PATH=/system/lib64:…`,
`dlopen("libvulkan.so")` from ggml-vulkan resolves to the Android loader,
which in turn discovers the vendor ICD. No `VK_ICD_FILENAMES` manifest
required, no chroot, no root. See `llm/server.go:StartRunner`.
