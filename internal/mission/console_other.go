//go:build !windows

package mission

import "os/exec"

func hideConsole(*exec.Cmd) {}
