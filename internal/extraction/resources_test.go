package extraction

import "testing"

func TestConservativeResourceSelection(t *testing.T) {
	for _, test := range []struct {
		name       string
		r          Resources
		total, ocr int
	}{
		{"unknown memory", Resources{Logical: 32, Physical: 16, AvailableCPUs: 32}, 1, 1},
		{"desktop", Resources{Logical: 16, Physical: 8, AvailableCPUs: 16, AvailableMemoryBytes: 16 << 30}, 5, 2},
		{"memory pressure", Resources{Logical: 16, Physical: 8, AvailableCPUs: 16, AvailableMemoryBytes: 2 << 30}, 2, 1},
		{"container quota", Resources{Logical: 32, Physical: 16, AvailableCPUs: 2, AvailableMemoryBytes: 16 << 30}, 1, 1},
		{"unknown physical cores", Resources{Logical: 8, AvailableCPUs: 8, AvailableMemoryBytes: 16 << 30}, 2, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := Recommend(test.r)
			if c.Total != test.total || c.OCR != test.ocr {
				t.Fatalf("unexpected limits %+v", c)
			}
		})
	}
	r := DetectResources()
	if r.Logical < 1 || r.AvailableCPUs < 1 || r.AvailableCPUs > r.Logical {
		t.Fatalf("bad detected resources: %+v", r)
	}
	t.Logf("detected resources: %+v; selected: %+v", r, Recommend(r))
	if (Concurrency{Mode: "manual", Native: 8, OCR: 1, Total: 2}).Validate() == nil {
		t.Fatal("inconsistent manual limit accepted")
	}
}
