package extraction

import (
	"fmt"
	"runtime"
)

type Resources struct {
	Logical              int      `json:"logical_cpus"`
	Physical             int      `json:"physical_cores"`
	AvailableCPUs        int      `json:"available_cpus"`
	MemoryBytes          uint64   `json:"memory_bytes"`
	AvailableMemoryBytes uint64   `json:"available_memory_bytes"`
	Notes                []string `json:"notes"`
}
type Concurrency struct {
	Mode   string `json:"mode"`
	Native int    `json:"native_workers"`
	OCR    int    `json:"ocr_workers"`
	Total  int    `json:"total_workers"`
}

func (c Concurrency) Validate() error {
	if c.Mode != "auto" && c.Mode != "manual" {
		return fmt.Errorf("selecciona configuración automática o manual")
	}
	if c.Mode == "manual" && (c.Native < 1 || c.Native > 8 || c.OCR < 1 || c.OCR > 4 || c.Total < 1 || c.Total > 8 || c.Native > c.Total || c.OCR > c.Total) {
		return fmt.Errorf("extracción: 1–8; OCR: 1–4; límite general: 1–8 y no menor que cada grupo")
	}
	return nil
}
func DetectResources() Resources {
	r := Resources{Logical: runtime.NumCPU(), AvailableCPUs: min(runtime.NumCPU(), runtime.GOMAXPROCS(0)), Notes: []string{}}
	detectPlatformResources(&r)
	r.AvailableCPUs = max(1, r.AvailableCPUs)
	if r.Physical == 0 {
		r.Notes = append(r.Notes, "No se pudo determinar el número de núcleos físicos; se aplica un límite conservador.")
	}
	if r.AvailableMemoryBytes == 0 {
		r.Notes = append(r.Notes, "Memoria disponible desconocida; se limita la concurrencia automática.")
	}
	return r
}
func Recommend(r Resources) Concurrency {
	cores := max(1, r.AvailableCPUs)
	if r.Physical > 0 {
		cores = min(cores, r.Physical)
	} else {
		cores = max(1, cores/2)
	}
	// Reserve CPU and RAM for the desktop, scanner, SQLite and other applications.
	total := max(1, min(6, (cores*2)/3))
	if r.AvailableMemoryBytes == 0 {
		total = 1
	} else {
		total = min(total, max(1, int((r.AvailableMemoryBytes/2)/(512<<20))))
	}
	ocr := max(1, min(3, total/2))
	return Concurrency{Mode: "auto", Native: total, OCR: ocr, Total: total}
}
