//go:build windows

package player

import (
	"os/exec"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	// Windows manages process groups differently. Returning nil is safe,
	// or we could use CreationFlags = 0x08000000 (CREATE_NO_WINDOW).
	return nil
}

func killProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
