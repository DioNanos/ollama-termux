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
| No auto-approve flags for CLI agents | Flags injected automatically (see Security Notice) |
| `xdg-open` fails on Termux | Uses `termux-open-url` for browser/OAuth |
| RAM detection reports ~1 GB instead of ~11 GB | Android-aware heuristic: 60-75% of MemTotal |
| All 8 cores used (thermal throttling on big.LITTLE) | Dynamic big core detection via cpufreq |
| No flash attention on mobile | Auto-enabled for memory savings |
| Cloud models shown in TUI (unusable on mobile) | Cloud models kept — best option on CPU-only devices |
| Context window too large for mobile RAM | Auto-limited per available memory tier |

---

## Security Notice

This fork automatically injects the following flags when launching Claude Code and Codex:

- `--dangerously-skip-permissions` (Claude Code)
- `--dangerously-bypass-approvals-and-sandbox` (Codex)

These flags allow the CLI agents to run autonomously without manual approval prompts. This is intended for use in a Termux sandbox environment where the risk profile is acceptable. Do not use this fork with access to sensitive data without understanding the implications.

---

## Quick Start

### Install (Pre-built)

```bash
# On Termux
pkg update && pkg upgrade -y
pkg install nodejs-lts -y

npm install -g @mmmbuto/ollama-termux
```

### Install CLIs

```bash
# Claude Code
npm install -g @anthropic-ai/claude-code

# Codex (Termux fork)
npm install -g @mmmbuto/codex-cli-termux@latest
```

### Install (Manual / On-device Build)

```bash
pkg install golang nodejs-lts -y
git clone https://github.com/DioNanos/ollama-termux.git
cd ollama-termux
go build -o ollama -ldflags="-s -w" -trimpath .
cp ollama /data/data/com.termux/files/usr/bin/ollama
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

## Build & Deploy (Cross-Compile on MGM)

For optimized builds with all ggml backends:

```bash
# On MGM (Linux x86_64)
export NDK_ROOT=~/android-ndk/android-ndk-r27c
./scripts/build_termux.sh

# Deploy to phone
scp dist/ollama-termux-*-android-arm64.tar.gz pixel9:~/

# On Termux
cd /data/data/com.termux/files/usr
tar -xzf ~/ollama-termux-*-android-arm64.tar.gz
chmod +x bin/ollama
```

See [BUILDING.md](./docs/BUILDING.md) for full details.

---

## Supported Devices

| Device | SoC | RAM | ggml Variant | Status |
|--------|-----|-----|-------------|--------|
| Pixel 9 Pro | Tensor G4 (ARMv8.6) | 16 GB | armv8.6 | Primary target |
| Pixel 10 Pro | Tensor G5 (ARMv8.6) | 16 GB | armv8.6 | Supported |
| Samsung Galaxy S24+ | Snapdragon 8 Gen 3 | 12 GB | armv8.6 | Supported |
| Samsung Galaxy S25 Ultra | Snapdragon 8 Elite | 12-16 GB | armv8.6 | Supported |
| Any modern Android | ARM64, 8+ GB RAM | - | armv8.2 | Should work |

---

## Architecture

```
ollama-termux (Go binary, our fork)
  |
  +-- cmd/launch/          Only Claude + Codex integrations
  +-- discover/cpu_linux.go  Android RAM heuristic (60-75% MemTotal)
  +-- llm/server.go          Big core detection, FA auto-enable, context auto-limit
  |
  +-- /usr/lib/ollama/*.so   ggml CPU backends (optimized for ARMv8.2+)
       |-- libggml-cpu-android_armv8.0_1.so   (fallback)
       |-- libggml-cpu-android_armv8.2_1.so   (+dotprod +fp16)
       +-- libggml-cpu-android_armv8.6_1.so   (+dotprod +fp16 +i8mm +sve2)
```

### Inference Optimizations

- **Thread count**: Dynamic detection of big (performance) cores via `/sys/devices/system/cpu/cpu*/cpufreq/cpuinfo_max_freq`. Cores at >= 75% of max frequency are used. Falls back to `NumCPU()/2` if cpufreq unavailable.
- **Flash attention**: Auto-enabled for KV cache memory savings
- **Context window**: Auto-limited based on available RAM:
  - < 2 GB → 2048 tokens
  - < 4 GB → 4096 tokens
  - < 8 GB → 8192 tokens
  - < 12 GB → 16384 tokens
  - >= 12 GB → 32768 tokens
- **RAM detection**: Clamped between 60-75% of MemTotal (avoids both undercounting from Android caching and overcounting from foreground app pressure)
- **mmap**: Already disabled on CPU-only by upstream (correct for Termux)
- **Backend selection**: Automatic via HWCAP scoring in ggml C++ layer

---

## What This Fork Does

- Strips Ollama to only Claude Code + Codex integrations
- Fixes all Termux/Android compatibility issues
- Optimizes inference parameters for smartphone hardware
- Injects auto-approve flags for autonomous CLI operation
- Cross-compiles optimized ggml backends for ARMv8.2+ and ARMv8.6+

## What This Fork Does Not Do

- Modify the ggml/llama.cpp inference engine core
- Add new integrations or features
- Replace upstream Ollama for desktop/server use
- Support GPU inference (Vulkan TODO for future)

---

## Roadmap

- [x] Go binary fork with only Claude + Codex
- [x] Termux browser fix (`termux-open-url`)
- [x] RAM detection fix for Android
- [x] Dynamic big core thread detection
- [x] Inference parameter tuning (threads, FA, context)
- [x] Extended context window ladder (up to 32k on 12+ GB devices)
- [x] Pre-built install via npm (no on-device toolchain)
- [x] Cross-compile build script (`scripts/build_termux.sh`)
- [x] Optimized ggml `.so` for ARMv8.2+ and ARMv8.6+ (3 variants)
- [ ] Vulkan GPU support via Mali-G715 / Adreno
- [ ] GitHub Actions CI for automated builds

---

## Repository

- **GitHub (public)**: https://github.com/DioNanos/ollama-termux
- **VPS3 (develop)**: `ssh://dag@cloud.alpacalibre.com:41822/home/dag/git_repos/ollama-termux.git`
- **Upstream**: [ollama/ollama](https://github.com/ollama/ollama) (MIT license)

## License

This project is licensed under the MIT license.

- Original work: [Ollama](https://github.com/ollama/ollama) (MIT)
- Termux optimizations: DioNanos
