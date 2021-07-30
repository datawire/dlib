package dexec

import (
	"syscall"
)

func init() {
	sysProcAttrForNewGroup = syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
