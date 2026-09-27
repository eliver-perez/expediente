package extraction

import (
	"context"
	"golang.org/x/sys/unix"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func detectPlatformResources(r *Resources) {
	physical, _ := unix.SysctlUint32("hw.physicalcpu")
	r.Physical = int(physical)
	r.MemoryBytes, _ = unix.SysctlUint64("hw.memsize")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "/usr/bin/vm_stat").Output()
	if err != nil {
		return
	}
	pageSize := uint64(4096)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "page size of ") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if f == "of" && i+1 < len(fields) {
					if n, e := strconv.ParseUint(fields[i+1], 10, 64); e == nil {
						pageSize = n
					}
				}
			}
		}
		// Inactive pages are reclaimable; avoid counting wired/compressed memory.
		if strings.HasPrefix(line, "Pages free:") || strings.HasPrefix(line, "Pages inactive:") || strings.HasPrefix(line, "Pages speculative:") {
			parts := strings.SplitN(line, ":", 2)
			n, _ := strconv.ParseUint(strings.Trim(strings.TrimSpace(parts[1]), "."), 10, 64)
			r.AvailableMemoryBytes += n * pageSize
		}
	}
}
