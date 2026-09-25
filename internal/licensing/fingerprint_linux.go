package licensing

import "os"

func machineIdentifier() (string, error) {
	contents, err := os.ReadFile("/etc/machine-id")
	if os.IsNotExist(err) {
		contents, err = os.ReadFile("/var/lib/dbus/machine-id")
	}
	return string(contents), err
}
