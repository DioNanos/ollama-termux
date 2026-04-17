#!/usr/bin/env bash
# Benchmark ollama-termux inference on Android/Termux.
#
# Usage:
#   ./scripts/bench_termux.sh [model]
#
# Output: JSON report to stdout and bench/results_<device>_<date>.json
#
# Prerequisites:
#   - ollama serve running
#   - curl, jq

set -euo pipefail

MODEL="${1:-qwen3.5:0.6b}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$ROOT_DIR/bench"
RESULTS_FILE="$BENCH_DIR/results_$(uname -n 2>/dev/null || echo 'unknown')_$(date +%Y%m%d_%H%M%S).json"

# Prompts of increasing length
PROMPTS=(
    "What is 2+2? Answer with a single number."
    "Explain the concept of recursion in programming. Give a concrete example in Python."
    "Write a function that implements binary search on a sorted array. Include edge cases, type hints, and docstring. Then explain the time complexity."
)

# Ensure bench dir exists
mkdir -p "$BENCH_DIR"

# Check server
if ! curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "ERROR: ollama server not running. Start with: ollama serve &"
    exit 1
fi

# Ensure model is available
if ! ollama list 2>/dev/null | grep -q "$(echo "$MODEL" | cut -d: -f1)"; then
    echo "Pulling model $MODEL..."
    ollama pull "$MODEL"
fi

echo "=== ollama-termux benchmark ==="
echo "Model: $MODEL"
echo "Device: $(uname -n 2>/dev/null || echo 'unknown')"
echo "Date: $(date -Iseconds)"
echo ""

# Collect system info
TOTAL_MEM=$(grep MemTotal /proc/meminfo | awk '{print $2}')
AVAIL_MEM=$(grep MemAvailable /proc/meminfo | awk '{print $2}')
CPU_INFO=$(grep -c "^processor" /proc/cpuinfo)
BIG_CORES=0
for f in /sys/devices/system/cpu/cpu*/cpufreq/cpuinfo_max_freq; do
    [ -f "$f" ] || continue
    freq=$(cat "$f" 2>/dev/null || echo 0)
    BIG_CORES=$((BIG_CORES + 1))
done

echo "System: ${CPU_INFO} CPUs, ${TOTAL_MEM} kB total, ${AVAIL_MEM} kB available"
echo ""

# Run benchmarks
RESULTS=()
PROMPT_SIZES=("short" "medium" "long")

for i in "${!PROMPTS[@]}"; do
    PROMPT="${PROMPTS[$i]}"
    SIZE="${PROMPT_SIZES[$i]}"
    echo "--- Prompt: $SIZE ($(echo -n "$PROMPT" | wc -c) chars) ---"

    RESPONSE=$(curl -sf http://localhost:11434/api/generate \
        -d "$(jq -n --arg model "$MODEL" --arg prompt "$PROMPT" '{
            model: $model,
            prompt: $prompt,
            stream: false,
            options: { num_predict: 256 }
        }')")

    if [ -z "$RESPONSE" ]; then
        echo "  ERROR: empty response"
        RESULTS+=("{\"size\":\"$SIZE\",\"error\":\"empty response\"}")
        continue
    fi

    EVAL_COUNT=$(echo "$RESPONSE" | jq -r '.eval_count // 0')
    EVAL_DURATION=$(echo "$RESPONSE" | jq -r '.eval_duration // 0')
    PROMPT_EVAL_COUNT=$(echo "$RESPONSE" | jq -r '.prompt_eval_count // 0')
    PROMPT_EVAL_DURATION=$(echo "$RESPONSE" | jq -r '.prompt_eval_duration // 0')
    LOAD_DURATION=$(echo "$RESPONSE" | jq -r '.load_duration // 0')

    # Convert nanoseconds to seconds
    EVAL_SEC=$(echo "$EVAL_DURATION" | awk '{printf "%.3f", $1/1000000000}')
    PROMPT_EVAL_SEC=$(echo "$PROMPT_EVAL_DURATION" | awk '{printf "%.3f", $1/1000000000}')

    # Tokens per second
    PROMPT_TPS=0
    GENERATE_TPS=0
    if [ "$PROMPT_EVAL_SEC" != "0.000" ] && [ "$PROMPT_EVAL_SEC" != "0" ]; then
        PROMPT_TPS=$(echo "$PROMPT_EVAL_COUNT $PROMPT_EVAL_SEC" | awk '{printf "%.1f", $1/$2}')
    fi
    if [ "$EVAL_SEC" != "0.000" ] && [ "$EVAL_SEC" != "0" ]; then
        GENERATE_TPS=$(echo "$EVAL_COUNT $EVAL_SEC" | awk '{printf "%.1f", $1/$2}')
    fi

    echo "  Prompt eval: ${PROMPT_EVAL_COUNT} tokens in ${PROMPT_EVAL_SEC}s (${PROMPT_TPS} tok/s)"
    echo "  Generate:    ${EVAL_COUNT} tokens in ${EVAL_SEC}s (${GENERATE_TPS} tok/s)"
    echo "  Load:        $(echo "$LOAD_DURATION" | awk '{printf "%.1f", $1/1000000000}')s"
    echo ""

    RESULTS+=("$(jq -n \
        --arg size "$SIZE" \
        --argjson prompt_tokens "$PROMPT_EVAL_COUNT" \
        --argjson prompt_sec "$PROMPT_EVAL_SEC" \
        --argjson prompt_tps "$PROMPT_TPS" \
        --argjson gen_tokens "$EVAL_COUNT" \
        --argjson gen_sec "$EVAL_SEC" \
        --argjson gen_tps "$GENERATE_TPS" \
        '{
            size: $size,
            prompt_tokens: $prompt_tokens,
            prompt_eval_sec: $prompt_sec,
            prompt_tps: $prompt_tps,
            generate_tokens: $gen_tokens,
            generate_sec: $gen_sec,
            generate_tps: $gen_tps
        }')")
done

# Write JSON report
REPORT=$(jq -n \
    --arg model "$MODEL" \
    --arg device "$(uname -n 2>/dev/null || echo 'unknown')" \
    --arg date "$(date -Iseconds)" \
    --argjson total_mem "$TOTAL_MEM" \
    --argjson avail_mem "$AVAIL_MEM" \
    --argjson cpus "$CPU_INFO" \
    --argjson big_cores "$BIG_CORES" \
    --argjson results "$(printf '%s\n' "${RESULTS[@]}" | jq -s '.')" \
    '{
        model: $model,
        device: $device,
        date: $date,
        system: {
            total_mem_kb: $total_mem,
            avail_mem_kb: $avail_mem,
            cpus: $cpus,
            big_cores: $big_cores
        },
        results: $results
    }')

echo "$REPORT" | tee "$RESULTS_FILE"
echo ""
echo "Results saved to: $RESULTS_FILE"
