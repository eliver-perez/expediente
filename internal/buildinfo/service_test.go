package buildinfo

import (
	"reflect"
	"testing"
)

func TestInstalledServiceIdentity(t *testing.T) {
	for _, name := range []string{"AIDEX"} {
		args := []string{"C:\\José\\AIDEX\\aidex-server.exe", "serve", "--config", "C:\\María López\\Muñoz\\config.json", "--windows-service", name}
		actual, service, err := WindowsServiceArguments(args, "AIDEX")
		if err != nil || service != name || !reflect.DeepEqual(actual, args[:4]) {
			t.Fatalf("%s: %v %s %v", name, actual, service, err)
		}
	}
	for _, options := range [][]string{{"--windows-service"}, {"--windows-service", "unrelated"}, {"--windows-service", "AIDEX", "--windows-service", "OtherService"}} {
		if _, _, err := WindowsServiceArguments(append([]string{"app", "serve"}, options...), "AIDEX"); err == nil {
			t.Fatal("accepted", options)
		}
	}
	args := []string{"app", "doctor", "--config", "ñ.json"}
	if actual, name, err := WindowsServiceArguments(args, "AIDEX"); err != nil || name != "AIDEX" || !reflect.DeepEqual(actual, args) {
		t.Fatal(actual, name, err)
	}
}
