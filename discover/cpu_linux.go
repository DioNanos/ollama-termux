package discover

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/format"
)

func GetCPUMem() (memInfo, error) {
	mem, err := getCPUMem()
	if err != nil {
		return memInfo{}, err
	}
	mem = getCPUMemByCgroups(mem)
	mem = adjustMemForAndroid(mem)
	return mem, nil
}

// adjustMemForAndroid converts Linux MemAvailable into a conservative Ollama
// loading budget on Termux. MemAvailable is kept as the upper bound: raising it
// to a percentage of total RAM can hide real Android pressure and invite LMKD
// or the kernel OOM killer. We reserve memory for Android and other apps, and
// reduce the budget further when zram/swap is nearly exhausted.
func adjustMemForAndroid(mem memInfo) memInfo {
	if !envconfig.IsTermux() && os.Getenv("PREFIX") != "/data/data/com.termux/files/usr" {
		return mem
	}

	budget := mem.FreeMemory
	if mem.TotalMemory > 0 {
		// A defensive ceiling protects against broken vendor MemAvailable
		// accounting without ever increasing the kernel-reported value.
		ceil := mem.TotalMemory - mem.TotalMemory/4
		if budget > ceil {
			budget = ceil
		}

		// Keep at least 1 GiB, or 12.5% on larger phones, outside Ollama's
		// loading budget for Android, Termux and foreground applications.
		headroom := mem.TotalMemory / 8
		if headroom < format.GibiByte {
			headroom = format.GibiByte
		}
		budget = saturatingSub(budget, headroom)
	}

	// Android commonly uses compressed zram. Less than 10% free swap is an
	// observable pressure signal even when MemAvailable has not caught up.
	if mem.TotalSwap > 0 && mem.FreeSwap < mem.TotalSwap/10 {
		budget /= 2
	}

	mem.FreeMemory = budget
	return mem
}

func saturatingSub(value, subtract uint64) uint64 {
	if subtract >= value {
		return 0
	}
	return value - subtract
}

func getCPUMem() (memInfo, error) {
	var mem memInfo
	var total, available, free, buffers, cached, totalSwap, freeSwap uint64
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return mem, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			_, err = fmt.Sscanf(line, "MemTotal:%d", &total)
		case strings.HasPrefix(line, "MemAvailable:"):
			_, err = fmt.Sscanf(line, "MemAvailable:%d", &available)
		case strings.HasPrefix(line, "MemFree:"):
			_, err = fmt.Sscanf(line, "MemFree:%d", &free)
		case strings.HasPrefix(line, "Buffers:"):
			_, err = fmt.Sscanf(line, "Buffers:%d", &buffers)
		case strings.HasPrefix(line, "Cached:"):
			_, err = fmt.Sscanf(line, "Cached:%d", &cached)
		case strings.HasPrefix(line, "SwapTotal:"):
			_, err = fmt.Sscanf(line, "SwapTotal:%d", &totalSwap)
		case strings.HasPrefix(line, "SwapFree:"):
			_, err = fmt.Sscanf(line, "SwapFree:%d", &freeSwap)
		default:
			continue
		}
		if err != nil {
			return mem, err
		}
	}
	mem.TotalMemory = total * format.KibiByte
	mem.TotalSwap = totalSwap * format.KibiByte
	mem.FreeSwap = freeSwap * format.KibiByte
	if available > 0 {
		mem.FreeMemory = available * format.KibiByte
	} else {
		mem.FreeMemory = (free + buffers + cached) * format.KibiByte
	}
	return mem, nil
}

func getCPUMemByCgroups(mem memInfo) memInfo {
	total, err := getUint64ValueFromFile("/sys/fs/cgroup/memory.max")
	if err != nil || total == 0 {
		return mem
	}

	used, err := getUint64ValueFromFile("/sys/fs/cgroup/memory.current")
	if err != nil {
		if mem.TotalMemory == 0 || total < mem.TotalMemory {
			mem.TotalMemory = total
		}
		if mem.FreeMemory > total {
			mem.FreeMemory = total
		}
		return mem
	}

	return constrainMemoryByCgroup(mem, total, used)
}

// constrainMemoryByCgroup applies a finite cgroup limit as an upper bound.
// A cgroup must never make host memory appear larger or underflow when usage
// races above the reported limit.
func constrainMemoryByCgroup(mem memInfo, total, used uint64) memInfo {
	if total == 0 {
		return mem
	}

	cgroupFree := saturatingSub(total, used)
	if mem.TotalMemory == 0 || total < mem.TotalMemory {
		mem.TotalMemory = total
	}
	if cgroupFree < mem.FreeMemory {
		mem.FreeMemory = cgroupFree
	}
	return mem
}

func getUint64ValueFromFile(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		return strconv.ParseUint(line, 10, 64)
	}
	return 0, errors.New("empty file content")
}

func IsNUMA() bool {
	ids := map[string]any{}
	packageIds, _ := filepath.Glob("/sys/devices/system/cpu/cpu*/topology/physical_package_id")
	for _, packageId := range packageIds {
		id, err := os.ReadFile(packageId)
		if err == nil {
			ids[strings.TrimSpace(string(id))] = struct{}{}
		}
	}
	return len(ids) > 1
}
