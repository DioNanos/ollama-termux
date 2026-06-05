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
- Ships prebuilt Android ARM64 release assets through GitHub Releases and
  installs them through the npm package

### Termux-Specific Runtime

- Inference runs through the upstream `llama-server` subprocess (Ollama
  0.30.x architecture), cross-built for Android ARM64
- RAM detection: 60-75% of MemTotal (Android-aware heuristic)
- Thread limit: big cores only (cpufreq-based detection)
- Context window: auto-limited based on available RAM tiers
- Library paths: `$PREFIX/lib` + `/system/lib64` (Android Vulkan loader)
  wired into the `llama-server` subprocess

---

## Installation

```bash
pkg update && pkg upgrade -y
pkg install nodejs-lts -y

# Current release line (0.30.x, llama-server runtime) — next channel
npm install -g @mmmbuto/ollama-termux@next
ollama-termux   # run the installer once

# Previous stable line (0.24.x)
npm install -g @mmmbuto/ollama-termux@latest
```

The `next` tag carries the current 0.30.x release line while it completes
device validation; `latest` stays on the previous line until promotion.

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
| 1 | **Codex** | `@mmmbuto/codex-cli-termux` | Termux fork |
| 2 | **Codex VL** | `@mmmbuto/codex-vl` | Vivling-enhanced fork |
| 3 | **Qwen Code** | `@mmmbuto/qwen-code-termux` | Termux fork |
| 4 | **Pi** | `@earendil-works/pi-coding-agent` | Upstream npm, Termux-compatible |

The launcher offers to install a missing integration when you select it
(npm-based, with confirmation). Manual install also works:

```bash
# Codex — our Termux fork
npm install -g @mmmbuto/codex-cli-termux

# Codex VL — our Vivling fork
npm install -g @mmmbuto/codex-vl

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

# Interactive menu: pick chat or one of the CLIs
ollama

# Or launch an integration directly
ollama launch codex --model qwen3.5:4b
ollama launch codex-vl --model gemma4:e4b
ollama launch qwen --model qwen3.5:4b
ollama launch pi
```

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

Tuned for modern ARM64 phones:

- Pixel 9 Pro / Tensor G4
- Galaxy S24+ / Snapdragon 8 Gen 3
- Galaxy S25 Ultra / Snapdragon 8 Elite

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
