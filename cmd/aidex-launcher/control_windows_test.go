package main

import (
	"reflect"
	"testing"
)

func TestControlCenterOperationArguments(t *testing.T) {
	if got, ok := controlOperation([]string{"Port", "18274"}); !ok || !reflect.DeepEqual(got, []string{"-Operation", "Port", "-Port", "18274"}) {
		t.Fatalf("port operation: %v, %v", got, ok)
	}
	for _, input := range [][]string{{}, {"Configure"}, {"Setup", "-File"}, {"Port", "65536"}, {"Port", "123 -File sample"}, {"Start", "--config", "user.json"}} {
		if _, ok := controlOperation(input); ok {
			t.Errorf("allowed unknown operation arguments %q", input)
		}
	}
}
