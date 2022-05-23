package dexec_test

import (
	"syscall"
)

func init() {
	newInterruptableSysProcAttr = func() *syscall.SysProcAttr {
		return &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}
	}
}
