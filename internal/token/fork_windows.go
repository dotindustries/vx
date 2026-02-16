//go:build windows

package token

import "syscall"

const (
	createNewProcessGroup = 0x00000200
	createNoWindow        = 0x08000000
)

// daemonSysProcAttr returns Windows-specific process attributes that place the
// child in a new process group without a console window, so it survives parent
// exit and runs silently in the background.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup | createNoWindow}
}
