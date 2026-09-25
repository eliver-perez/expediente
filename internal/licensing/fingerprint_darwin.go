package licensing

import (
	"context"
	"os/exec"
	"regexp"
	"time"
)

func machineIdentifier() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := exec.CommandContext(ctx, "/usr/sbin/ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return "", err
	}
	match := regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([0-9A-Fa-f-]+)"`).FindSubmatch(result)
	if len(match) != 2 {
		return "", contractFailure()
	}
	return string(match[1]), nil
}
