package extraction

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func numberFile(path string) uint64 {
	b, _ := os.ReadFile(path)
	n, _ := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	return n
}
func detectPlatformResources(r *Resources) {
	var affinity unix.CPUSet
	if unix.SchedGetaffinity(0, &affinity) == nil && affinity.Count() > 0 {
		r.AvailableCPUs = min(r.AvailableCPUs, affinity.Count())
	}
	info, _ := os.ReadFile("/proc/cpuinfo")
	cores := map[string]bool{}
	for _, processor := range strings.Split(string(info), "\n\n") {
		socket, core := "", ""
		for _, line := range strings.Split(processor, "\n") {
			p := strings.SplitN(line, ":", 2)
			if len(p) != 2 {
				continue
			}
			switch strings.TrimSpace(p[0]) {
			case "physical id":
				socket = strings.TrimSpace(p[1])
			case "core id":
				core = strings.TrimSpace(p[1])
			}
		}
		if socket != "" && core != "" {
			cores[socket+":"+core] = true
		}
	}
	r.Physical = len(cores)
	memory, _ := os.ReadFile("/proc/meminfo")
	for _, line := range strings.Split(string(memory), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		n, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			r.MemoryBytes = n * 1024
		case "MemAvailable:":
			r.AvailableMemoryBytes = n * 1024
		}
	}
	// Traverse cgroup v2 ancestors, including a namespace-mounted root. Quotas of
	// parent slices also apply; unavailable v1 data is reported, never guessed.
	groups, _ := os.ReadFile("/proc/self/cgroup")
	found := false
	for _, line := range strings.Split(string(groups), "\n") {
		if !strings.HasPrefix(line, "0::") {
			continue
		}
		found = true
		root := "/sys/fs/cgroup"
		dir := filepath.Join(root, strings.TrimPrefix(strings.TrimPrefix(line, "0::"), "/"))
		for {
			cpu, _ := os.ReadFile(filepath.Join(dir, "cpu.max"))
			f := strings.Fields(string(cpu))
			if len(f) == 2 && f[0] != "max" {
				quota, _ := strconv.Atoi(f[0])
				period, _ := strconv.Atoi(f[1])
				if quota > 0 && period > 0 {
					r.AvailableCPUs = min(r.AvailableCPUs, max(1, quota/period))
				}
			}
			limit := numberFile(filepath.Join(dir, "memory.max"))
			used := numberFile(filepath.Join(dir, "memory.current"))
			if limit > 0 {
				r.MemoryBytes = min(r.MemoryBytes, limit)
				available := uint64(1)
				if used < limit {
					available = limit - used
				}
				r.AvailableMemoryBytes = min(r.AvailableMemoryBytes, available)
			}
			if dir == root || dir == "/" {
				break
			}
			dir = filepath.Dir(dir)
		}
	}
	if !found {
		r.Notes = append(r.Notes, "No se detectó cgroup v2; se aplican afinidad de CPU y memoria del sistema.")
	}
}
