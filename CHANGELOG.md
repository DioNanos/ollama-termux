# Changelog

## 0.33.1-termux — 2026-08-28

This EOL Termux release aligns the fork with upstream Ollama `v0.33.1` for
Android ARM64. It preserves the fork-owned CI pipeline, Termux launcher and
prebuilt archive verification while incorporating upstream runtime changes.

### Termux changes

- Keep the Android ARM64 release workflow with NDK checksum validation,
  verify-only default, archive validation and CI-built `llama-server` assets.
- Preserve Termux launcher/runtime compatibility and the npm installer safety
  contract.
- Remove the obsolete UI `LaunchCommands` component superseded upstream by the
  shared `launchCommands` integration registry.

This is the final EOL alignment release; no further upstream synchronization is
promised.

### Upstream highlights

- Claude recommendation mappings and updated agent/runtime behavior from
  upstream Ollama `v0.33.1`.
- Updated model, parser, server and llama.cpp support carried by the upstream
  release.

## 0.32.2-termux.1 — 2026-07-22

This stable Termux release merges upstream Ollama `v0.32.2` (24 commits since
`v0.32.1`) on top of the `0.32.1-termux.1` baseline (`a8d284a54`).

### Termux changes

- Merge upstream `v0.32.2`. Two textual conflicts were resolved:
  - `.github/workflows/release.yaml` — kept the fork-owned Termux release
    workflow (NDK input checksum, exact stable-tag binding, verify-only
    default, archive verification before publish). Upstream's multi-platform
    macOS/Windows/CUDA/ROCm release matrix does not apply to Android ARM64.
  - `cmd/cmd.go` — accepted upstream's `checkServerHeartbeat` server-start
    refactor (`#17245`/`#17229`/`#17227`), which removed the former
    `ensureServerRunning` helper and the `background_unix`/`background_windows`
    `SysProcAttr` helpers.
- Preserve the fork's Termux auto-start of `ollama serve`. Upstream routed
  root-command/TUI server start through `checkServerHeartbeat` -> `startApp`;
  on non-darwin/windows hosts `startApp` previously only returned an actionable
  error. `cmd/start_default.go` now launches `ollama serve` in the background
  through the Android dynamic linker when `TermuxSystemLinkerExec()` requires
  it (Play-Store SELinux domain), with `Setpgid` detach and a heartbeat wait,
  restoring the behaviour the removed `ensureServerRunning` provided. A unit
  test covers the non-Termux default error path.
- Make the auto-start wait fail closed: the client now honors context
  cancellation and returns an actionable timeout after 15 seconds instead of
  waiting forever when the background server exits before becoming healthy.
  Deterministic tests cover success, timeout and cancellation.
- Carry forward the two mobile hardening fixes from `0.32.1-termux.1`,
  verified intact after the merge:
  - Vulkan loader order: `/system/lib64` precedes Termux library directories
    in `termuxRunnerLibraryPaths()` so Mesa/llvmpipe cannot shadow the Android
    vendor GPU loader (regression test `TestTermuxRunnerLibraryPaths`).
  - Conservative Android memory budget: `adjustMemForAndroid()` never raises
    Linux `MemAvailable`, reserves Android headroom, respects cgroup limits
    and backs off under zram/swap pressure (regression tests
    `TestAdjustMemForAndroidConservativeBudget`, `TestConstrainMemoryByCgroup`).
- Keep the verified Termux launcher surface (Codex VL, Codex, Qwen Code, Pi);
  upstream's Hermes integration update (`#17202`) and Claude Code channels
  (`#17210`) merged into the shared launch code but stay gated unsupported on
  Termux by `cmd/launch/termux.go`.

### Upstream highlights (v0.32.2)

- Agent skills system, agent semantics/UX/DX cleanup, unlimited tool rounds
  for cloud models, and removal of the standalone agent command and dead
  agent prompt wrappers.
- `server`: detect download stalls before the first byte (`#17259`).
- `model`: Laguna v8 chat support and Metal inference fix (`#17291`).
- `llama.cpp` update (`#17186`); MLX update (`#17189`).
- Linux toolchain bumped to GCC 13 (`#17244`); Windows ARM64 CUDA support
  (`#16931`) — not exercised on Android but merged cleanly.

### Release channel and device compatibility

This version is published as the stable npm `latest` release. The Android
ARM64 archive, Vulkan payload, installer safety and Termux regression suites
are verified in CI. Vulkan behavior remains dependent on the Android vendor
driver: affected Adreno 650 devices may need `OLLAMA_VULKAN=0` for CPU fallback
while [issue #1](https://github.com/DioNanos/ollama-termux/issues/1) remains
under investigation. Additional device validation continues after release.

### Mobile model note

The `0.32.2` base can pull models requiring a newer Ollama runtime, including
`ornith:9b`. On phones its advertised 256K context must not be treated as a
usable target; use the fork's automatic context clamp and validate under real
device memory pressure. `ornith:35b` is outside the supported phone envelope.

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
