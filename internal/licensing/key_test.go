package licensing

import "testing"

func TestCommercialKeyNormalization(t *testing.T) {
	for _, input := range []string{"K7M-9R3X-6TVP-8CHD", "k7m9r3x6tvp8chd", "  k7m-9r3x-6tvp-8chd  "} {
		got, err := NormalizeCommercialKey(input)
		if err != nil || got != "K7M-9R3X-6TVP-8CHD" {
			t.Fatalf("%q: %q %v", input, got, err)
		}
	}
	for _, input := range []string{"AIDEX-K7M-9R3X-6TVP-8CHD", "K7M-9R3X-6TVP-8CH0", "K7M9-R3X6-TVP8-CHD", "K7M 9R3X 6TVP 8CHD", "Ｋ7M-9R3X-6TVP-8CHD", "K7M-9R3X-6TVP-8CHD-"} {
		if _, err := NormalizeCommercialKey(input); err == nil {
			t.Fatalf("invalid code accepted: %q", input)
		}
	}
}
