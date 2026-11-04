//go:build darwin && cgo

package main

import (
	"strings"
	"testing"
)

func TestTrustedServicePIDAndListenerValidation(t *testing.T) {
	pid, err := registeredPID("system/app.aidex = {\n state = running\n pid = 3127\n}")
	if err != nil || pid != "3127" {
		t.Fatal(pid, err)
	}
	for _, output := range []string{"state = waiting", "pid = 0", "pid = -2", "otherpid = 3127"} {
		if _, err := registeredPID(output); err == nil {
			t.Fatal("se aceptó un servicio detenido o desconocido", output)
		}
	}
	if !listenerMatches("3127\n", "3127") {
		t.Fatal("listener válido rechazado")
	}
	for _, output := range []string{"", "9999\n", "3127\n9999\n", "3127;comando\n", "p3127\nf3\n"} {
		if listenerMatches(output, "3127") {
			t.Fatal("se aceptó un listener ajeno o ambiguo", output)
		}
	}
}

func TestTicketURLRejectsArbitraryHostsOperationsAndArguments(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	valid := "aidex-auth://authorize/" + nonce + "?port=18090"
	if endpoint, err := ticketEndpoint(valid); err != nil || endpoint != "http://127.0.0.1:18090/api/v1/local-folder-authorizations/"+nonce {
		t.Fatal(endpoint, err)
	}
	for _, invalid := range []string{"https://example.com/" + nonce, "aidex-auth://authorize.evil/" + nonce + "?port=18090", valid + "&command=rm", valid + "&port=80", strings.ReplaceAll(valid, "18090", "0"), strings.ReplaceAll(valid, "18090", "65536"), strings.ReplaceAll(valid, "18090", "018090"), valid + "#otro", strings.ReplaceAll(valid, "authorize/", "user@authorize/"), strings.ReplaceAll(valid, nonce, "../../tmp"), strings.ReplaceAll(valid, nonce, nonce+"%0A")} {
		if _, err := ticketEndpoint(invalid); err == nil {
			t.Fatalf("enlace peligroso aceptado: %q", invalid)
		}
	}
}
