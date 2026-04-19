# Benchmarks — ollama-termux v0.21.0-termux.14

On-device inference benchmarks for the Termux/Android ARM64 build,
comparing CPU-only vs. Vulkan GPU offload (via the Android system loader).

## Test rig

- **Device**: Pixel 9 Pro (Tensor G4)
- **GPU**: Mali-G715 (iGPU, reported 15.1 GiB shared memory)
- **OS**: Android 14 / Termux
- **Build**: `@mmmbuto/ollama-termux@0.21.0-termux.14`
- **Backend**: ggml-vulkan via `/system/lib64/libvulkan.so`
- **Prompt**: ~55 token Italian prompt, `num_predict=400`, `seed=42`
- **Context**: 4096, BatchSize 512, Threads 4
- **CPU mode**: FlashAttention disabled (default)
- **GPU mode**: FlashAttention enabled, `OLLAMA_VULKAN=1`

## Results

| Model             | Size    | Mode   | Layers | Load (s) | Eval (s) | **tok/s** | Speedup |
|-------------------|---------|--------|-------:|---------:|---------:|----------:|--------:|
| `gemma4:e2b`      | 6.83 GB | CPU    |    — / 36 |     9.97 |   163.64 | **2.44**  |    1.0× |
| `gemma4:e2b`      | 6.83 GB | Vulkan |  36/36 |    93.08 |    54.25 | **7.37**  |  **3.02×** |
| `medgemma:latest` | 3.18 GB | CPU    |    — / 35 |    14.67 |   164.65 | **2.43**  |    1.0× |
| `medgemma:latest` | 3.18 GB | Vulkan |  35/35 |    39.97 |   100.96 | **3.96**  |  **1.63×** |
| `gemma4:e4b`      | 9.16 GB | CPU    |    — / 43 |    48.28 |   221.51 | **1.81**  |    1.0× |
| `gemma4:e4b`      | 9.16 GB | Vulkan |  43/43 |   122.84 |    89.83 | **4.45**  |  **2.46×** |

All three models achieve **100% layer offload** on the Mali-G715 with
`OLLAMA_VULKAN=1`. No CPU fallback, no silent downgrade.

## Observations

- **Vulkan wins on every model**, with the largest gain on `gemma4:e2b`
  (3× speedup). Smaller quant-mix models benefit less (medgemma 1.6×)
  likely because their tensor shapes fit CPU cache well.
- **Vulkan trades load time for eval throughput.** GPU load is
  ~2–10× slower (kernel warmup, tensor upload) but once resident the
  per-token cost is a fraction of CPU. Worth it for any session with
  more than a few hundred tokens of generation.
- **`gemma4:e4b` on CPU is borderline unusable** at 1.81 tok/s; Vulkan
  makes it viable for real work.
- **Memory pressure is real.** Android killed Termux twice during the
  benchmark run (between `medgemma` and `e4b` in both modes). Keep
  `OLLAMA_KEEP_ALIVE` reasonable and watch `free -h`.

## Methodology

```bash
# CPU mode
ollama serve &

# Vulkan mode
OLLAMA_VULKAN=1 ollama serve &

# Single-shot benchmark via REST (same prompt for both modes)
curl -s http://127.0.0.1:11434/api/generate -d '{
  "model": "<name>",
  "prompt": "<~55 tok IT prompt>",
  "stream": false,
  "options": {"num_predict": 400, "seed": 42}
}'
```

Throughput is computed from the response envelope:

```
tok/s = eval_count / (eval_duration / 1e9)
```

## Reproducing

```bash
npm i -g @mmmbuto/ollama-termux
OLLAMA_VULKAN=1 ollama serve
ollama pull gemma4:e2b
# then issue the /api/generate request shown above
```

Verify Vulkan discovery in the server log:

```
inference compute library=Vulkan name=Vulkan0 description=Mali-G715 \
  total="15.1 GiB" available="15.1 GiB"
```

and GPU offload for each loaded model:

```
offloaded N/N layers to GPU
```

## Changelog context

- `v0.21.0-termux.14` — fixed `LibOllamaPath` for `GOOS=android` so the
  runner subprocess finds `lib/ollama/vulkan/*.so` and actually loads the
  Vulkan backend. Prior `.12` / `.13` builds silently fell back to CPU.
