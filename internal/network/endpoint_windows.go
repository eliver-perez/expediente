//go:build windows

package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// The installer creates this one public, read-only-for-users file. Keeping its
// inode/ACL while writing lets the service publish a new URL without granting
// ordinary users access to the private configuration or installation directory.
func publishEndpoint(configurationPath, origin string) error {
	path := filepath.Join(filepath.Dir(configurationPath), "connection.json")
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("invalid public endpoint file")
	}
	contents, err := json.Marshal(map[string]string{"public_url": origin, "service_name": "AIDEX"})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = file.Write(append(contents, '\n')); err != nil {
		return err
	}
	return file.Sync()
}
