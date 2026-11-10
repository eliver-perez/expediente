package libraries

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// InspectRestoreDirectory reuses the exact identity encoding required by
// openLinked, so a restored root can be read without weakening identity checks.
func InspectRestoreDirectory(path string) (string, string, bool, error) {
	info, err := inspectDirectory(path)
	return info.Path, info.Identity, info.CaseSensitive, err
}

func InspectRestoreFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	identity, _, err := physicalIdentity(file)
	return identity, err
}

// VerifyRestoreReference uses the same confined, link-rejecting open path as
// normal document reads before making an imported file available.
func VerifyRestoreReference(rootPath, rootIdentity, relative, expectedHash string, expectedSize int64) (string, error) {
	if expectedHash == "" || expectedSize < 0 {
		return "", fmt.Errorf("reference has no verifiable content version")
	}
	file, err := openLinked(Root{Path: rootPath, Identity: rootIdentity}, relative)
	if err != nil {
		return "", err
	}
	defer file.Close()
	identity, _, err := physicalIdentity(file)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(file, expectedSize+1))
	if err != nil || size != expectedSize || hex.EncodeToString(hash.Sum(nil)) != expectedHash {
		return "", fmt.Errorf("reference content does not match the backup")
	}
	return identity, nil
}
