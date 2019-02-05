package virtualbox

import (
	"fmt"
	"strings"
)

func (vb *VBox) DisableDHCPServer(netName string) (string, error) {
	_, err := vb.manage("dhcpserver", "remove", "--netname", netName)
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		return "", err
	}
	return "Disabled dhcp server", nil
}

func (vb *VBox) EnableDHCPServer(netName string, ip string, netmask string, lowerIP string, upperIP string) (string, error) {
	return vb.manage("dhcpserver", "add", "--netname", netName, fmt.Sprintf("--ip=%s", ip), fmt.Sprintf("--netmask=%s", netmask), fmt.Sprintf("--lowerip=%s", lowerIP), fmt.Sprintf("--upperip=%s", upperIP), "--enable")
}
