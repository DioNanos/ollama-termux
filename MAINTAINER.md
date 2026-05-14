# Maintainer

Ollama Termux is maintained by **Davide A. Guglielmi** (GitHub:
[DioNanos](https://github.com/DioNanos)) as the porting / distribution
maintainer for Android ARM64 (Termux).

This is **not** an independent fork — Ollama Termux tracks
[ollama/ollama](https://github.com/ollama/ollama) and adds a Termux-first
distribution flow with mobile-oriented runtime tuning. Upstream Ollama source
tree and MIT license are preserved.

## Scope of maintenance

In scope:

- Termux-specific runtime tuning (CPU thread selection, memory heuristics,
  flash attention defaults, context-window limits per RAM tier)
- the `LD_LIBRARY_PATH` fix for runner subprocesses on Termux
- Vulkan loader path for `/system/lib64` (Android GPU access)
- the `@mmmbuto/ollama-termux` npm installer wrapper that downloads matching
  GitHub Release assets and installs them under the Termux prefix
- launcher integrations on Termux: **Codex VL** (primary), Codex, Qwen Code,
  Claude Code (frozen), Hermes Agent
- the release line: `v<upstream>-termux.N`, prebuilt Android ARM64 assets via
  GitHub Releases

Out of scope here:

- changes that belong upstream — please file those on
  [ollama/ollama](https://github.com/ollama/ollama) directly
- features unrelated to Termux / Android ARM64 packaging

## Reporting

| Channel | Where |
|---|---|
| Termux/Android bug reports, PRs | [DioNanos/ollama-termux](https://github.com/DioNanos/ollama-termux) |
| Generic Ollama bugs (not Termux-specific) | [ollama/ollama](https://github.com/ollama/ollama) |
| Security disclosures (Termux fork) | [`SECURITY.md`](./SECURITY.md) — `security@mmmbuto.com` |
| General contact | `dev@mmmbuto.com` |

When reporting a Termux bug, please include: device, Android version, Termux
build, total RAM, `ollama --version`, and the failing CLI integration if any
(Codex VL, Codex, Qwen Code, Claude Code, Hermes).

## Identity

- Profile: [github.com/DioNanos](https://github.com/DioNanos)
- Project hub: [mmmbuto.com](https://mmmbuto.com)
- Maintainer page and dev journal: [dev.mmmbuto.com](https://dev.mmmbuto.com)

## License

Ollama Termux is distributed under the MIT license inherited from
[ollama/ollama](https://github.com/ollama/ollama). The Termux distribution
work is released under the same license. Original upstream:
[ollama/ollama](https://github.com/ollama/ollama). Termux fork work:
DioNanos.
See [`LICENSE`](./LICENSE).

---

*Per aspera ad astra.*
