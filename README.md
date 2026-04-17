# Ollama Termux

> Smartphone-optimized Ollama fork for **Termux / Android ARM64**.
> Only **Claude Code** and **Codex** integrations — tested and verified on real devices.
> Maximum inference optimization for modern smartphones (Pixel / Samsung Galaxy).

[![platform](https://img.shields.io/badge/platform-Android%20ARM64-green?style=flat-square&logo=android)](https://termux.dev)
[![go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![license](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](./LICENSE)
[![upstream](https://img.shields.io/badge/upstream-ollama%20latest-blue?style=flat-square)](https://github.com/ollama/ollama)

---

## Why This Fork

Official Ollama works on Termux but has issues on modern smartphones:

| Problem | Fix |
|---------|-----|
| 10 integrations, most untested on Android | Only Claude + Codex, verified working |
| Codex rejects `@mmmbuto/codex-cli-termux` version | Accepts Termux fork with `-termux` suffix |
| No auto-approve flags for CLI agents | `--dangerously-skip-permissions` / `--dangerously-bypass-approvals-and-sandbox` injected automatically |
| `xdg-open` fails on Termux | Uses `termux-open-url` for browser/OAuth |
| RAM detection reports ~1 GB instead of ~11 GB | Android-aware heuristic: 70% of MemTotal |
| All 8 cores used (thermal throttling on big.LITTLE) | Thread limit to 5 (big cores only) |
| No flash attention on mobile | Auto-enabled for memory savings |
| Cloud models shown in TUI (unusable on mobile) | Cloud models kept — best option on CPU-only devices |

---

## Quick Start

### Prerequisites

```bash
pkg update && pkg upgrade -y
pkg install golang nodejs-lts -y
```

### Install CLIs

```bash
# Claude Code
npm install -g @anthropic-ai/claude-code

# Codex (Termux fork)
npm install -g @mmmbuto/codex-cli-termux@latest
```

### Build Ollama-Termux

```bash
git clone https://github.com/DioNanos/ollama-termux.git
cd ollama-termux
go build -o ollama-termux -ldflags="-s -w" -trimpath .
```

### Deploy

```bash
# Backup original
cp /data/data/com.termux/files/usr/bin/ollama /data/data/com.termux/files/usr/bin/ollama.orig

# Install fork
cp ollama-termux /data/data/com.termux/files/usr/bin/ollama
chmod +x /data/data/com.termux/files/usr/bin/ollama
```

---

## Usage

```bash
# Start server
ollama serve &

# Pull a model
ollama pull qwen3.5

# Launch Claude Code with auto-approve
ollama launch claude --model qwen3.5

# Launch Codex with auto-approve
ollama launch codex --model qwen3.5
```

---

## Supported Devices

| Device | SoC | RAM | Status |
|--------|-----|-----|--------|
| Pixel 9 Pro | Tensor G4 (ARMv8.2+) | 16 GB | Primary target |
| Samsung Galaxy S24+ | Snapdragon 8 Gen 3 | 12 GB | Supported |
| Samsung Galaxy S25 Ultra | Snapdragon 8 Elite | 12 GB | Supported |
| Any modern Android | ARM64, 8+ GB RAM | - | Should work |

---

## Architecture

```
ollama-termux (Go binary, our fork)
  |
  +-- cmd/launch/          Only Claude + Codex integrations
  +-- discover/cpu_linux.go  Android RAM fix
  +-- llm/server.go          Thread limit, FA auto-enable, context auto-limit
  |
  +-- /usr/lib/ollama/*.so   ggml CPU backends (optimized for ARMv8.2+)
       |-- libggml-cpu-android_armv8.0_1.so   (fallback)
       |-- libggml-cpu-android_armv8.2_1.so   (Pixel 9 Pro uses this)
       +-- libggml-cpu-android_armv8.6_1.so   (SVE/SVE2 devices)
```

### Inference Optimizations

- **Thread count**: Limited to 5 on mobile (big cores only, no LITTLE core overhead)
- **Flash attention**: Auto-enabled for KV cache memory savings
- **Context window**: Auto-limited based on available RAM (2048/4096/8192 tokens)
- **RAM detection**: 70% of MemTotal heuristic for Android (fixes the 1 GB vs 11 GB issue)
- **mmap**: Already disabled on CPU-only by upstream (correct for Termux)
- **Backend selection**: Automatic via HWCAP scoring in ggml C++ layer

### ggml Cross-Compilation

For maximum performance, ggml should be cross-compiled with:
- `-march=armv8.2-a+dotprod+i8mm+fp16`
- LTO enabled
- SVE/SVE2 kernels

See [BUILDING.md](./docs/BUILDING.md) for cross-compilation instructions.

---

## What This Fork Does

- Strips Ollama to only Claude Code + Codex integrations
- Fixes all Termux/Android compatibility issues
- Optimizes inference parameters for smartphone hardware
- Injects auto-approve flags for autonomous CLI operation

## What This Fork Does Not Do

- Modify the ggml/llama.cpp inference engine (yet)
- Add new integrations or features
- Replace upstream Ollama for desktop/server use
- Support GPU inference (Vulkan TODO for future)

---

## Roadmap

- [x] Go binary fork with only Claude + Codex
- [x] Termux browser fix (`termux-open-url`)
- [x] RAM detection fix for Android
- [x] Inference parameter tuning (threads, FA, context)
- [x] Cross-compile optimized ggml `.so` for ARMv8.2+
- [ ] Vulkan GPU support via Mali G715
- [ ] GitHub public release under MIT license

---

## Repository

- **GitHub (public)**: https://github.com/DioNanos/ollama-termux
- **VPS3 (develop)**: `ssh://dag@cloud.alpacalibre.com:41822/home/dag/git_repos/ollama-termux.git`
- **Upstream**: [ollama/ollama](https://github.com/ollama/ollama) (MIT license)

## License

This project is licensed under the MIT license.

- Original work: [Ollama](https://github.com/ollama/ollama) (MIT)
- Termux optimizations: DioNanos
