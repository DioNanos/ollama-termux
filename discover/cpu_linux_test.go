package discover

import (
	"testing"

	"github.com/ollama/ollama/format"
)

func TestAdjustMemForAndroidOffTermuxIsNoop(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("PREFIX", "/tmp/not-termux")
	want := memInfo{
		TotalMemory: 16 * format.GibiByte,
		FreeMemory:  3 * format.GibiByte,
		TotalSwap:   4 * format.GibiByte,
		FreeSwap:    128 * format.MebiByte,
	}
	if got := adjustMemForAndroid(want); got != want {
		t.Fatalf("adjustMemForAndroid() = %+v, want %+v", got, want)
	}
}

func TestAdjustMemForAndroidConservativeBudget(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.0")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")

	tests := []struct {
		name string
		mem  memInfo
		want uint64
	}{
		{
			name: "never inflates low MemAvailable",
			mem:  memInfo{TotalMemory: 16 * format.GibiByte, FreeMemory: 3 * format.GibiByte},
			want: 1 * format.GibiByte,
		},
		{
			name: "caps implausibly high availability then reserves Android headroom",
			mem:  memInfo{TotalMemory: 16 * format.GibiByte, FreeMemory: 15 * format.GibiByte},
			want: 10 * format.GibiByte,
		},
		{
			name: "swap pressure halves remaining budget",
			mem: memInfo{
				TotalMemory: 16 * format.GibiByte,
				FreeMemory:  3 * format.GibiByte,
				TotalSwap:   4 * format.GibiByte,
				FreeSwap:    128 * format.MebiByte,
			},
			want: 512 * format.MebiByte,
		},
		{
			name: "headroom subtraction saturates",
			mem:  memInfo{TotalMemory: 8 * format.GibiByte, FreeMemory: 512 * format.MebiByte},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adjustMemForAndroid(tt.mem)
			if got.FreeMemory != tt.want {
				t.Fatalf("FreeMemory = %d, want %d", got.FreeMemory, tt.want)
			}
			if got.FreeMemory > tt.mem.FreeMemory {
				t.Fatalf("FreeMemory increased from %d to %d", tt.mem.FreeMemory, got.FreeMemory)
			}
		})
	}
}

func TestConstrainMemoryByCgroup(t *testing.T) {
	base := memInfo{TotalMemory: 16 * format.GibiByte, FreeMemory: 8 * format.GibiByte}
	tests := []struct {
		name  string
		input memInfo
		total uint64
		used  uint64
		want  memInfo
	}{
		{
			name:  "finite limit reduces total and available",
			input: base,
			total: 12 * format.GibiByte,
			used:  10 * format.GibiByte,
			want:  memInfo{TotalMemory: 12 * format.GibiByte, FreeMemory: 2 * format.GibiByte},
		},
		{
			name:  "larger cgroup values never inflate host values",
			input: base,
			total: 20 * format.GibiByte,
			used:  1 * format.GibiByte,
			want:  base,
		},
		{
			name:  "usage above limit saturates",
			input: base,
			total: 4 * format.GibiByte,
			used:  5 * format.GibiByte,
			want:  memInfo{TotalMemory: 4 * format.GibiByte, FreeMemory: 0},
		},
		{
			name:  "zero host availability is never inflated",
			input: memInfo{TotalMemory: 16 * format.GibiByte, FreeMemory: 0},
			total: 12 * format.GibiByte,
			used:  1 * format.GibiByte,
			want:  memInfo{TotalMemory: 12 * format.GibiByte, FreeMemory: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constrainMemoryByCgroup(tt.input, tt.total, tt.used); got != tt.want {
				t.Fatalf("constrainMemoryByCgroup() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
