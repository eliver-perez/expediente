package folderpicker

import (
	"strings"
	"testing"
)

func TestEndpointOnlyAcceptsOpaqueLoopbackSelection(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	valid := "aidex-pick://select/" + nonce + "?port=18090"
	got, err := Endpoint(valid)
	if err != nil || got != "http://127.0.0.1:18090/api/v1/folder-selections/"+nonce {
		t.Fatal(got, err)
	}
	for _, raw := range []string{strings.Replace(valid, "select/", "select.evil/", 1), valid + "&port=80", valid + "&path=/etc", valid + "#x", strings.Replace(valid, nonce, "../../etc", 1), strings.Replace(valid, "18090", "018090", 1), strings.Replace(valid, "18090", "65536", 1), strings.Replace(valid, "select/", "user@select/", 1), "https://select/" + nonce + "?port=18090"} {
		if _, err := Endpoint(raw); err == nil {
			t.Fatal("accepted", raw)
		}
	}
}
