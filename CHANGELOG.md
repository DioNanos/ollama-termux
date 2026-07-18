# Changelog

## 0.32.1-termux.1 — 2026-07-18

This stable Termux release is based on upstream Ollama `v0.32.1`.

### Termux changes

- Prefer `/system/lib64` over Termux library directories when resolving
  `libvulkan.so`, preventing Mesa/llvmpipe from shadowing Android's vendor GPU
  loader. The order is covered by regression tests and was A/B verified on a
  Pixel 9 Pro (Mali-G715).
- Replace the previous 60%-of-total RAM floor with a conservative budget that
  never raises Linux `MemAvailable`, reserves Android headroom, respects finite
  cgroup limits, and backs off under zram/swap pressure.
- Keep the verified Termux launcher surface: Codex VL, Codex, Qwen Code and Pi.
- Make the npm installer fail closed on missing or malformed SHA-256 metadata,
  reject archive traversal/links/special entries, and validate the full payload
  before replacing an installation.
- Bind release builds to the exact stable tag, checksum the Android NDK input,
  verify release archives again before npm publish, reject version collisions,
  and publish the stable package explicitly to npm `latest`.

### Upstream highlights

- New interactive agent UI with working-directory context.
- Improved Gemma 4 tool calling and multi-turn reasoning.
- Fixed recurrent MLX cache growth and improved cache performance.
- `OLLAMA_LOAD_TIMEOUT` support for MLX text model loading.
- Improved launcher handling for deprecated model selections and authenticated
  agent web search/fetch.

### Mobile model note

The 0.32.1 base can pull models requiring a newer Ollama runtime, including
`ornith:9b`. On phones its advertised 256K context must not be treated as a
usable target; use the fork's automatic context clamp and validate under real
device memory pressure. `ornith:35b` is outside the supported phone envelope.
