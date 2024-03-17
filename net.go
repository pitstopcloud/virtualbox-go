package virtualbox

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

func (vb *VBox) PortForwarding(vm *VirtualMachine, rule PortForwarding) error {
	_, err := vb.manage("modifyvm", vm.UUIDOrName(), fmt.Sprintf("--natpf%d", rule.Index), fmt.Sprintf("\"%v,%v,%v,%v,%v,%v\"", rule.Name, string(rule.Protocol), rule.HostIP, rule.HostPort, rule.GuestIP, rule.GuestPort))
	return err
}

func (vb *VBox) PortForwardingDelete(vm *VirtualMachine, index int, name string) error {
	_, err := vb.manage("modifyvm", vm.UUIDOrName(), fmt.Sprintf("--natpf%d", index), "delete", name)
	return err
}

func (vb *VBox) HostOnlyNetInfo() ([]Network, error) {
	out, err := vb.manage("list", "hostonlyifs")
	if err != nil {
		return nil, err
	}

	var nws []Network

	var nw Network
	_ = tryParseKeyValues(out, reColonLine, func(key, val string, ok bool) error {
		switch key {
		case "Name":
			nw.Name = val
		case "GUID":
			nw.GUID = val
		case "HardwareAddress":
			nw.HWAddress = val
		case "VBoxNetworkName":
			nw.DeviceName = val[len("HostInterfaceNetworking-"):]
		default:
			if !ok && strings.TrimSpace(val) == "" {
				nw.Mode = NWMode_hostonly
				nws = append(nws, nw)
				nw = Network{}
			}
		}
		return nil
	})
	return nws, nil
}

func (vb *VBox) NatNetInfo() ([]Network, error) {
	out, err := vb.manage("list", "natnets")
	if err != nil {
		return nil, err
	}

	var nws []Network

	var nw Network
	_ = tryParseKeyValues(out, reColonLine, func(key, val string, ok bool) error {
		switch key {
		case "NetworkName":
			nw.Name = val
		default:
			if !ok && strings.TrimSpace(val) == "" {
				nw.Mode = NWMode_natnetwork
				nws = append(nws, nw)
				nw = Network{}
			}
		}
		return nil
	})
	return nws, nil
}

func (vb *VBox) InternalNetInfo() ([]Network, error) {
	out, err := vb.manage("list", "intnets")
	if err != nil {
		return nil, err
	}

	var nws []Network

	var nw Network
	_ = tryParseKeyValues(out, reColonLine, func(key, val string, ok bool) error {
		switch key {
		case "Name":
			nw.Name = val
		default:
			if !ok && strings.TrimSpace(val) == "" {
				nw.Mode = NWMode_intnet
				nws = append(nws, nw)
				nw = Network{}
			}
		}
		return nil
	})
	return nws, nil
}

func (vb *VBox) BridgeNetInfo() ([]Network, error) {
	out, err := vb.manage("list", "bridgedifs")
	if err != nil {
		return nil, err
	}

	var nws []Network

	var nw Network
	_ = tryParseKeyValues(out, reColonLine, func(key, val string, ok bool) error {
		switch key {
		case "Name":
			nw.Name = val
		case "GUID":
			nw.GUID = val
		case "HardwareAddress":
			nw.HWAddress = val
		case "VBoxNetworkName":
			nw.DeviceName = val[len("HostInterfaceNetworking-"):]
		default:
			if !ok && strings.TrimSpace(val) == "" {
				nw.Mode = NWMode_bridged
				nws = append(nws, nw)
				nw = Network{}
			}
		}
		return nil
	})
	return nws, nil
}

func (vb *VBox) SyncNICs() (err error) {

	if hostOnlyNws, err := vb.HostOnlyNetInfo(); err != nil {
		return err
	} else {
		for i := range hostOnlyNws {
			vb.HostOnlyNws[hostOnlyNws[i].Name] = &hostOnlyNws[i]
		}
	}

	if internalNws, err := vb.InternalNetInfo(); err != nil {
		return err
	} else {
		for i := range internalNws {
			vb.InternalNws[internalNws[i].Name] = &internalNws[i]
		}
	}

	if natNws, err := vb.NatNetInfo(); err != nil {
		return err
	} else {
		for i := range natNws {
			vb.NatNws[natNws[i].Name] = &natNws[i]
		}
	}

	if bridgedNws, err := vb.BridgeNetInfo(); err != nil {
		return err
	} else {
		for i := range bridgedNws {
			vb.BridgedNws[bridgedNws[i].Name] = &bridgedNws[i]
		}
	}

	return nil
}

func (vb *VBox) CreateNet(net *Network) error {

	out, err := vb.manage("hostonlyif", "create")
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`Interface '([^']+)' was successfully created`)
	matches := re.FindStringSubmatch(out)
	if len(matches) >= 2 {
		net.Name = matches[1]
	} else {
		return fmt.Errorf("could not determine the interface name from vbox output: %s", out)
	}

	return err
}

func (vb *VBox) DeleteNet(net *Network) error {

	switch net.Mode {
	case NWMode_hostonly:
		_, err := vb.manage("hostonlyif", "remove", net.Name)
		if err != nil && isHostDeviceNotFound(err.Error()) {
			return NotFoundError(err.Error())
		}
	case NWMode_natnetwork:
		_, err := vb.manage("natnetwork", "remove", "--netname", net.Name)
		if err != nil && isHostDeviceNotFound(err.Error()) {
			return NotFoundError(err.Error())
		}
	} //others are no op

	return nil
}

func isHostDeviceNotFound(text string) bool {
	return strings.Contains(text, "could not be found")
}

func (vb *VBox) AddNic(vm *VirtualMachine, nic *NIC) error {
	args := []string{}
	switch nic.Mode {
	case NWMode_bridged:
		args = append(args, fmt.Sprintf("--nic %d", nic.Index), string(NWMode_bridged), fmt.Sprintf("--bridgeadapter%d", nic.Index), nic.NetworkName)
	case NWMode_hostonly:
		args = append(args, fmt.Sprintf("--nic%d", nic.Index), string(NWMode_hostonly), fmt.Sprintf("--hostonlyadapter%d", nic.Index), nic.NetworkName)
	case NWMode_intnet:
		args = append(args, fmt.Sprintf("--nic%d", nic.Index), string(NWMode_intnet), fmt.Sprintf("--intnet%d", nic.Index), nic.NetworkName)
	case NWMode_natnetwork:
		args = append(args, fmt.Sprintf("--nic%d", nic.Index), string(NWMode_natnetwork), fmt.Sprintf("--nat-network%d", nic.NetworkName))
	}

	args = append(args, fmt.Sprintf("--nictype%d", nic.Index), string(nic.Type))

	_, err := vb.modify(vm, args...)
	return err
}

func (vb *VBox) SetNICDefaults(vm *VirtualMachine) error {
	if err := vb.SyncNICs(); err != nil {
		return err
	}

	verrs := ValidationErrors{}

	nics := vm.Spec.NICs
	for i := range nics {
		//set defaults
		nics[i].Index = i + 1 // will override the set index value

		if nics[i].Mode == "" {
			nics[i].Mode = NWMode_hostonly
		}

		if nics[i].Type == "" {
			nics[i].Type = NIC_82540EM
		}

		if nics[i].NetworkName == "" {
			if nics[i].Mode == NWMode_intnet {
				verrs.Add(fmt.Sprintf("nic/%d", i), fmt.Errorf("networkname missing for internal net"))
				continue
			}

			if nw, err := vb.getDefaultNetwork(nics[i].Mode); err == nil {
				nics[i].NetworkName = nw.Name
			} else {
				verrs.Add(fmt.Sprintf("nic/%d", i), err)
				continue
			}
		}
	}

	if len(verrs.errors) > 0 {
		return verrs
	}
	return nil
}

func (vb *VBox) getNetwork(nw string, mode NetworkMode) (*Network, error) {
	switch mode {
	case NWMode_bridged:
		return vb.BridgedNws[nw], nil
	case NWMode_hostonly:
		return vb.HostOnlyNws[nw], nil
	case NWMode_intnet:
		return vb.InternalNws[nw], nil
	case NWMode_natnetwork:
		return vb.NatNws[nw], nil
	default:
		return nil, nil
	}
}

func (vb *VBox) getDefaultNetwork(mode NetworkMode) (*Network, error) {
	var nws []*Network
	switch mode {
	case NWMode_bridged:
		nws = make([]*Network, 0, len(vb.BridgedNws))
		for _, v := range vb.BridgedNws {
			nws = append(nws, v)
		}
	case NWMode_hostonly:
		nws = make([]*Network, 0, len(vb.HostOnlyNws))
		for _, v := range vb.HostOnlyNws {
			nws = append(nws, v)
		}
	case NWMode_intnet:
		nws = make([]*Network, 0, len(vb.InternalNws))
		for _, v := range vb.InternalNws {
			nws = append(nws, v)
		}
	case NWMode_natnetwork:
		nws = make([]*Network, 0, len(vb.NatNws))
		for _, v := range vb.NatNws {
			nws = append(nws, v)
		}
	}

	sort.Slice(nws, func(i, j int) bool {
		return nws[i].Name < nws[j].Name
	})

	if len(nws) > 0 {
		return nws[0], nil
	}

	return nil, nil
}

func (vb *VBox) EnsureNets() error {
	return nil
}
