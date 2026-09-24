package extraction

import "os/exec"

// Configured tools run directly and do not launch a shell or helper pipeline.
func configureProcess(command *exec.Cmd) {}
