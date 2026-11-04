//go:build darwin

package macosfolders

import "testing"

func TestSelectionRejectsBroadSystemAndEscapedPaths(t *testing.T) {
	for _, path := range []string{"/", "/Users", "/Users/Ana", "/Volumes", "/Volumes/Datos", "/Library/Application Support/AIDEX", "/Users/Ana/Library/secretos", "/Users/_aidex/documentos", "/Users/Ana/Documentos/../", "/Users/Ana//Documentos", "/Users/Ana/Documentos\ncomando", "/Users/Ana/Documentos\u202eotra", "/Users/Ana/Documentos\\otro", "relative"} {
		if err := ValidateSelection(Selection{Path: path, Source: "managed"}); err == nil {
			t.Fatalf("ruta aceptada: %q", path)
		}
	}
	for _, path := range []string{"/Users/Ana/Documentos/Obra 2026", "/Users/Shared/Biblioteca", "/Volumes/Documentos/Obra"} {
		if err := ValidateSelection(Selection{Path: path, Source: "linked"}); err != nil {
			t.Fatal(path, err)
		}
	}
	if err := ValidateSelection(Selection{Path: "/Users/Ana/docs", Source: "shell"}); err == nil {
		t.Fatal("operación arbitraria aceptada")
	}
}

func TestFingerprintBindsPathScopeAndFinderIdentity(t *testing.T) {
	selection := Selection{Path: "/Users/Ana/Documentos", Source: "linked"}
	expected := Fingerprint(selection, 501)
	if expected != Fingerprint(selection, 501) {
		t.Fatal("huella no determinista")
	}
	selection.Source = "managed"
	if expected == Fingerprint(selection, 501) {
		t.Fatal("se puede cambiar el alcance")
	}
	selection.Source = "linked"
	selection.Path += "/otra"
	if expected == Fingerprint(selection, 501) {
		t.Fatal("se puede cambiar la ruta")
	}
	selection.Path = "/Users/Ana/Documentos"
	if expected == Fingerprint(selection, 502) {
		t.Fatal("se puede cambiar el destinatario de Finder")
	}
}
