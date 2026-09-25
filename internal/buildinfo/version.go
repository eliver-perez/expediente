// Package buildinfo identifies a distributable without relying on its filename.
package buildinfo

const Version = "0.7.0-test.1"

// Set only by the installer build. Ordinary developer binaries keep their paths.
var Channel = "source"
var ServiceName = ""
