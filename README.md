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

- Keeps only the launcher integrations we support on Termux:
  **Codex VL** (primary), **Codex**, **Qwen Code**, **Claude Code** (frozen),
  **Hermes Agent**
- Uses `termux-open-url` for browser/OAuth flows
- Tunes CPU thread selection, memory heuristics, flash attention defaults, and
  context limits for modern phones
- Ships prebuilt Android ARM64 release assets through GitHub Releases and
  installs them through the npm package

### Termux-Specific Runtime

- RAM detection: 60-75% of MemTotal (Android-aware heuristic)
- Thread limit: big cores only (cpufreq-based detection)
- Flash attention: auto-enabled on CPU-only for memory savings
- Context window: auto-limited based on available RAM tiers
- Vulkan: `/system/lib64` loader path for Android GPU access
- `LD_LIBRARY_PATH` fix for runner subprocess on Termux

---

## Installation

```bash
pkg update && pkg upgrade -y
pkg install nodejs-lts -y

npm install -g @mmmbuto/ollama-termux@0.23.2-termux.3
```

The npm package is an installer wrapper. During `postinstall`, it downloads the
matching GitHub Release asset, verifies SHA256, and installs `bin/ollama` +
`lib/ollama/*.so` under the Termux prefix.

---

## Supported Integrations

| Order | CLI | Package | Status |
|-------|-----|---------|--------|
| 1 | **Codex VL** | `@mmmbuto/codex-vl` | Primary — Vivling-enhanced fork |
| 2 | **Codex** | `@mmmbuto/codex-cli-termux` | Secondary |
| 3 | **Qwen Code** | `@mmmbuto/qwen-code-termux` | OpenAI-compat via local Ollama |
| 4 | **Claude Code** | `@anthropic-ai/claude-code@2.1.112` | Frozen (Anthropic dropped Termux) |
| 5 | **Hermes Agent** | curl install script | Official Termux support |

Install the CLIs you need:

```bash
# Codex VL — primary (our Vivling fork)
npm install -g @mmmbuto/codex-vl

# Codex — secondary (our Termux fork)
npm install -g @mmmbuto/codex-cli-termux

# Qwen Code
npm install -g @mmmbuto/qwen-code-termux

# Claude Code — frozen, do NOT upgrade past 2.1.112
npm install -g @anthropic-ai/claude-code@2.1.112

# Hermes Agent
curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash
```

---

## Quickstart

```bash
# Start Ollama
ollama serve &

# Pull recommended local models
ollama pull qwen3.5:4b
ollama pull gemma4:e4b

# Launch integrations
ollama launch codex-vl --model gemma4:e4b
ollama launch codex --model qwen3.5:4b
ollama launch qwen --model gemma4:e2b
ollama launch claude --model qwen3.5:4b
ollama launch hermes

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

CPU backends: `armv8.0`, `armv8.2+dotprod+fp16`, `armv8.6+dotprod+fp16+i8mm+sve2`.
Optional Vulkan GPU backend (`BUILD_VULKAN=1`, runtime `OLLAMA_VULKAN=1`).

---

## Links

- npm: https://www.npmjs.com/package/@mmmbuto/ollama-termux
- Releases: https://github.com/DioNanos/ollama-termux/releases
- Upstream: https://github.com/ollama/ollama

---

## License

MIT — original upstream [ollama/ollama](https://github.com/ollama/ollama).
Termux fork work: DioNanos.
