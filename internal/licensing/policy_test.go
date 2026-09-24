package licensing

import "testing"

func TestSecurityRecoveryAlwaysAvailableAndDocumentGateRespectsBuild(t *testing.T) {
	if err := Check(SecurityAdministration); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []Operation{ReadDocuments, WriteDocuments} {
		if allowed := Check(operation) == nil; allowed != DevelopmentEnabled() {
			t.Fatal("development policy leaked across build boundary")
		}
	}
}
