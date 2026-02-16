//go:build !windows

package token

import "syscall"

// daemonSysProcAttr returns Unix-specific process attributes that create a new
// session, detaching the child from the parent's process group.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
