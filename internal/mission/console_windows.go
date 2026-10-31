package mission

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// hideConsole keeps a console window from flashing up over the game.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
