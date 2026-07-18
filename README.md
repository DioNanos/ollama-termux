# Ollama Termux

> Built from upstream [Ollama](https://github.com/ollama/ollama), adapted as a
> Termux-first fork for Android ARM64 devices.

[![npm](https://img.shields.io/npm/v/@mmmbuto/ollama-termux?style=flat-square&logo=npm)](https://www.npmjs.com/package/@mmmbuto/ollama-termux)
[![platform](https://img.shields.io/badge/platform-Android%20ARM64-3DDC84?style=flat-square&logo=android&logoColor=white)](https://termux.dev)
[![license](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](./LICENSE)

---

## What This Is

`ollama-termux` is an explicit fork of upstream Ollama for Termux on modern
Android ARM64 phones. It keeps the upstream Ollama codebase and release naming
scheme, but adds a Termux-specific distribution flow and mobile-oriented runtime
behavior.

### What We Keep From Upstream

- Upstream Ollama source tree and MIT license
- Upstream version lineage, published as `v<upstream>-termux.N`
- Standard `ollama` CLI and server behavior

### What This Fork Changes

- Exposes only the launcher integrations verified on Termux:
  **Codex**, **Codex VL**, **Qwen Code**, **Pi**
- Auto-installs missing integrations from npm straight from the launcher menu
- Uses `termux-open-url` for browser/OAuth flows
- Tunes CPU thread selection, memory heuristics, and context limits for
  modern phones
- Resolves Vulkan through Android's system loader before Termux Mesa, avoiding
  silent llvmpipe/CPU fallback
- Ships prebuilt Android ARM64 release assets through GitHub Releases and
  installs them through the npm package

### Termux-Specific Runtime

- Inference runs through the upstream `llama-server` subprocess (Ollama
  0.32.x architecture), cross-built for Android ARM64
- RAM budget: never exceeds Linux `MemAvailable`; reserves Android headroom
  and backs off further when zram/swap is nearly exhausted
- Thread limit: big cores only (cpufreq-based detection)
- Context window: auto-limited based on available RAM tiers
- Library paths: `/system/lib64` before `$PREFIX/lib`, wired into the
  `llama-server` subprocess so the vendor driver wins over Termux Mesa/llvmpipe

---

## Installation

```bash
pkg update && pkg upgrade -y
pkg install nodejs-lts -y

npm install -g @mmmbuto/ollama-termux@latest
ollama-termux   # run the installer once
```

The npm package is an installer wrapper: `ollama-termux` downloads the
matching GitHub Release asset, verifies SHA256, and installs `bin/ollama` +
the `lib/ollama` runtime (llama-server + backend libraries) under the Termux
prefix. Recent npm versions block `postinstall` scripts by default
(allow-scripts), so running `ollama-termux` after the install is the
reliable path; on older npm the postinstall hook does the same thing
automatically.

---

## Supported Integrations

| Order | CLI | Package | Status |
|-------|-----|---------|--------|
| 1 | **Codex VL** | `@mmmbuto/codex-vl` | Vivling-enhanced fork — **primary on Termux** |
| 2 | **Codex** | `@mmmbuto/codex-cli-termux` | Termux fork |
| 3 | **Qwen Code** | `@mmmbuto/qwen-code-termux` | Termux fork |
| 4 | **Pi** | `@earendil-works/pi-coding-agent` | Upstream npm, Termux-compatible |

On Termux, Codex VL is the primary integration: it is listed first in the
menu, and a bare `ollama` / `ollama launch` on a fresh install (no prior
selection) drops straight into Codex VL. If Codex VL is not installed, the
menu is shown instead.

The launcher offers to install a missing integration when you select it
(npm-based, with confirmation). Manual install also works:

```bash
# Codex VL — our Vivling fork (primary on Termux)
npm install -g @mmmbuto/codex-vl

# Codex — our Termux fork
npm install -g @mmmbuto/codex-cli-termux

# Qwen Code — our Termux fork
npm install -g @mmmbuto/qwen-code-termux

# Pi
npm install -g @earendil-works/pi-coding-agent
```

---

## Quickstart

```bash
# Start Ollama
ollama serve &

# Pull recommended local models
ollama pull qwen3.5:4b
ollama pull gemma4:e4b

# On Termux, first run drops straight into Codex VL (primary);
# otherwise pick chat or a CLI from the menu
ollama

# Or launch an integration directly
ollama launch codex-vl --model gemma4:e4b
ollama launch codex --model qwen3.5:4b
ollama launch qwen --model qwen3.5:4b
ollama launch pi
```

### Larger phone models

`ornith:9b` can be pulled by this Ollama base and is the largest Ornith variant
worth testing on a 16 GB phone. Its advertised 256K context is not a mobile
target: the fork clamps context from the conservative live-memory budget. Start
with the default/4K context and `OLLAMA_VULKAN=1`; stop if Android begins heavy
zram swapping. `ornith:35b` is not a supported phone target.

---

## Build

```bash
export NDK_ROOT=~/android-ndk/android-ndk-r27c
./scripts/build_termux.sh
```

Output: `dist/ollama-termux-<version>-android-arm64.tar.gz`

See [docs/BUILDING.md](./docs/BUILDING.md) for the full cross-build flow.

---

## Devices

Runtime-validated device:

- Pixel 9 Pro / Tensor G4 / Mali-G715 (CPU and Android-system-loader Vulkan)

Additional ARM64 targets covered by runtime-dispatched CPU variants and the
Android Vulkan path, but still requiring release-candidate device validation:

- ASUS ROG Phone 3 / Snapdragon 865+ / Adreno 650
- Galaxy S24+ / Snapdragon 8 Gen 3 / Adreno
- Galaxy S25 Ultra / Snapdragon 8 Elite / Adreno

CPU backends: runtime-dispatched llama.cpp variant libraries
(`GGML_CPU_ALL_VARIANTS`), selected per device at startup.
Optional Vulkan GPU backend (`BUILD_VULKAN=1`).

---

## Links

- npm: https://www.npmjs.com/package/@mmmbuto/ollama-termux
- Releases: https://github.com/DioNanos/ollama-termux/releases
- Upstream: https://github.com/ollama/ollama

---

## License

MIT — original upstream [ollama/ollama](https://github.com/ollama/ollama).
Termux fork work: DioNanos.

---

## Contact

Maintained by [DioNanos](https://github.com/DioNanos).

- General / dev: **dev@mmmbuto.com**
- Security disclosures: **security@mmmbuto.com**
- Project hub: <https://mmmbuto.com>
