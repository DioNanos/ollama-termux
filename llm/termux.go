package llm

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/ml"
)

// Termux/Android runtime tuning for the ollama-termux fork. All helpers are
// no-ops off Termux so upstream behavior is preserved everywhere else.

// limitTermuxContext clamps the context window to what phone memory can hold.
// Android reports aggressive-caching-adjusted free memory (see
// discover.adjustMemForAndroid); the tiers below keep the KV cache from
// triggering the oom_killer on 8-16 GB devices.
func limitTermuxContext(opts *api.Options, systemInfo ml.SystemInfo) {
	if !envconfig.IsTermux() {
		return
	}

	memMB := systemInfo.FreeMemory / (1024 * 1024)
	var maxCtx int
	switch {
	case memMB < 2048:
		maxCtx = 2048
	case memMB < 4096:
		maxCtx = 4096
	case memMB < 8192:
		maxCtx = 8192
	case memMB < 12288:
		maxCtx = 16384
	default:
		maxCtx = 32768
	}
	if opts.NumCtx > maxCtx {
		slog.Info("termux: limiting context window", "requested", opts.NumCtx, "limited", maxCtx, "free_mem_mb", memMB)
		opts.NumCtx = maxCtx
	}
}

// termuxDefaultThreads returns the llama-server thread count for Termux when
// the user did not set one. big.LITTLE phones throttle when LITTLE cores join
// the worker pool, so inference is pinned to the performance cores.
func termuxDefaultThreads() int {
	if !envconfig.IsTermux() {
		return 0
	}
	return countBigCores()
}

// countBigCores returns the number of performance cores on Android by reading
// cpufreq max frequency. Cores with freq >= 75% of the highest frequency are
// considered big cores. Falls back to half of total CPUs if cpufreq is
// unavailable.
func countBigCores() int {
	entries, err := filepath.Glob("/sys/devices/system/cpu/cpu*/cpufreq/cpuinfo_max_freq")
	if err != nil || len(entries) == 0 {
		n := runtime.NumCPU() / 2
		if n < 1 {
			n = 1
		}
		return n
	}
	freqs := make([]int64, 0, len(entries))
	var maxFreq int64
	for _, p := range entries {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		f, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
		if err != nil {
			continue
		}
		freqs = append(freqs, f)
		if f > maxFreq {
			maxFreq = f
		}
	}
	if maxFreq == 0 {
		return runtime.NumCPU() / 2
	}
	threshold := int64(float64(maxFreq) * 0.75)
	big := 0
	for _, f := range freqs {
		if f >= threshold {
			big++
		}
	}
	if big < 1 {
		big = 1
	}
	return big
}

// termuxRunnerCommand builds the llama-server invocation. On Play-Store
// Termux (targetSdk >= 29) SELinux denies direct execve of app data files
// and Go bypasses the termux-exec libc shim, so the subprocess is started
// through the system linker.
func termuxRunnerCommand(exe string, params []string) *exec.Cmd {
	if envconfig.TermuxSystemLinkerExec() {
		return exec.Command(envconfig.TermuxSystemLinker, append([]string{exe}, params...)...)
	}
	return exec.Command(exe, params...)
}

// termuxRunnerLibraryPaths returns extra library directories for the
// llama-server subprocess on Termux:
//   - $PREFIX/lib for libc++_shared.so and the Termux-built ggml libraries
//   - /system/lib64 so dlopen("libvulkan.so") can reach the Android system
//     Vulkan loader, which runs in a linker namespace with access to the
//     vendor GPU driver
func termuxRunnerLibraryPaths() []string {
	if !envconfig.IsTermux() {
		return nil
	}

	prefixes := []string{os.Getenv("PREFIX"), "/data/data/com.termux/files/usr"}
	seen := map[string]bool{}
	var paths []string
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		lib := filepath.Join(prefix, "lib")
		if !seen[lib] {
			seen[lib] = true
			paths = append(paths, lib)
		}
	}
	return append(paths, "/system/lib64")
}
