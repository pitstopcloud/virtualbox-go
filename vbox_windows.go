// +build windows

package virtualbox

import (
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func vboxManagePath() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Oracle\VirtualBox`, registry.QUERY_VALUE)
	if err != nil {
		return VBoxManage
	}
	defer k.Close()

	s, _, err := k.GetStringValue("InstallDir")
	if err != nil {
		return VBoxManage
	}
	return filepath.Join(s, VBoxManage)
}
