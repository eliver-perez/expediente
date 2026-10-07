// Package buildinfo identifies a distributable without relying on its filename.
package buildinfo

const Version = "2.0.0-alpha.8"

// Set only by the installer build. Ordinary developer binaries keep their paths.
var Channel = "source"
var ServiceName = ""

// Packaging-only revision; data compatibility and license protocol version stay unchanged.
var Revision = ""

func DisplayVersion() string {
	if Revision != "" {
		return Version + "-" + Revision
	}
	return Version
}
