// Package macoscontrol publishes the small, local status view that can be read
// when AIDEX is stopped. The service configuration remains private.
package macoscontrol

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"gestor-documental/internal/buildinfo"
)

func Endpoint(configPath, address, origin string) {
	_, rawPort, err := net.SplitHostPort(address)
	if err == nil {
		port, _ := strconv.Atoi(rawPort)
		write(configPath, "endpoint.json", struct {
			Port int    `json:"port"`
			URL  string `json:"url"`
		}{port, origin})
	}
}

func Database(configPath, state, setup string) {
	if runtime.GOOS != "darwin" || buildinfo.Channel != "production" {
		return
	}
	contents, err := os.ReadFile(filepath.Join(state, "postgresql", "installation.json"))
	if err != nil {
		return
	}
	var private struct {
		Port int `json:"port"`
	}
	if json.Unmarshal(contents, &private) == nil {
		write(configPath, "database.json", struct {
			Port  int    `json:"port"`
			Setup string `json:"setup"`
		}{private.Port, setup})
	}
}

func write(configPath, name string, value any) {
	if runtime.GOOS != "darwin" || buildinfo.Channel != "production" ||
		filepath.Base(configPath) != "config.json" || filepath.Base(filepath.Dir(configPath)) != "data" {
		return
	}
	base := filepath.Dir(filepath.Dir(configPath))
	if filepath.Base(base) != "AIDEX" {
		return
	}
	directory := filepath.Join(base, "control")
	if info, err := os.Lstat(directory); err != nil || !info.IsDir() {
		return
	}
	file, err := os.CreateTemp(directory, ".status-*")
	if err != nil {
		return
	}
	defer os.Remove(file.Name())
	err = json.NewEncoder(file).Encode(value)
	if err == nil {
		err = file.Chmod(0644)
	}
	if closeErr := file.Close(); err == nil && closeErr == nil {
		_ = os.Rename(file.Name(), filepath.Join(directory, name))
	}
}
