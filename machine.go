package virtualbox

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (vb *VBox) CreateVM(vm *VirtualMachine) error {

	args := []string{"createvm", "--name", vm.Spec.Name, "--ostype", vm.Spec.OSType.ID}

	//args = append(args, "--basefolder", strconv.Quote(vm.Path))
	args = append(args, "--basefolder", vb.Config.BasePath)

	if vm.Spec.Group != "" {
		args = append(args, "--groups", vm.Spec.Group)
	}

	_, err := vb.manage(args...)

	if err != nil && isAlreadyExistErrorMessage(err.Error()) {
		return AlreadyExistsErrorr.New(vb.getVMSettingsFile(vm))
	}

	// TODO: Get the UUID populated in the vm.UUID
	return err
}

// DeleteVM removes the setting file and must be  used with caution.  The VM must be unregistered before calling this
func (vb *VBox) DeleteVM(vm *VirtualMachine) error {
	return os.RemoveAll(vb.getVMSettingsFile(vm))
}

// TODO: Ensure this is idempotent
func (vb *VBox) RegisterVM(vm *VirtualMachine) error {
	_, err := vb.manage("registervm", vb.getVMSettingsFile(vm))
	return err
}

func (vb *VBox) UnRegisterVM(vm *VirtualMachine) error {
	_, err := vb.manage("unregistervm", vb.getVMSettingsFile(vm))
	return err
}

func (vb *VBox) AddStorageController(vm *VirtualMachine, ctr StorageController) error {

	_, err := vb.manage("storagectl", vm.UUIDOrName(), "--name", ctr.Name, "--add", string(ctr.Type))
	if err != nil && isAlreadyExistErrorMessage(err.Error()) {
		return AlreadyExists(vm.Spec.Name)
	}
	return nil
}

func (vb *VBox) AttachStorage(vm *VirtualMachine, disk *Disk) error {
	nonRotational := "off"
	if disk.NonRotational {
		nonRotational = "on"
	}
	autoDiscard := "off"
	if disk.AutoDiscard {
		if disk.Format != VDI {
			glog.Warning(
				"Disk format ", disk.Format, " is not VDI. ",
				"Ignoring AutoDiscard.")
		} else {
			autoDiscard = "on"
		}
	}
	_, err := vb.manage(
		"storageattach", vm.Spec.Name,
		"--storagectl", disk.Controller.Name,
		"--port", strconv.Itoa(disk.Controller.Port),
		"--device", strconv.Itoa(disk.Controller.Device),
		"--type", string(disk.Type),
		"--medium", disk.Path,
		"--nonrotational", nonRotational,
		"--discard", autoDiscard)

	return err
}

func (vb *VBox) SetMemory(vm *VirtualMachine, sizeMB int) error {
	_, err := vb.modify(vm, "--memory", strconv.Itoa(sizeMB))
	return err
}

func (vb *VBox) SetCPUCount(vm *VirtualMachine, cpus int) error {
	_, err := vb.modify(vm, "--cpus", strconv.Itoa(cpus))
	return err
}

func (vb *VBox) SetBootOrder(vm *VirtualMachine, bootOrder []BootDevice) error {
	args := []string{}
	for i, b := range bootOrder {
		args = append(args, fmt.Sprintf("--boot%d", i+1), string(b))
	}
	_, err := vb.modify(vm, args...)
	return err
}

func (vb *VBox) Start(vm *VirtualMachine) (string, error) {
	return vb.manage("startvm", vm.UUIDOrName(), "--type", "headless")
}

func (vb *VBox) Stop(vm *VirtualMachine) (string, error) {
	return vb.control(vm, "poweroff")
}

func (vb *VBox) Restart(vm *VirtualMachine) (string, error) {
	vb.Stop(vm)
	return vb.Start(vm)
}

func (vb *VBox) Save(vm *VirtualMachine) (string, error) {
	return vb.control(vm, "save")
}

func (vb *VBox) Pause(vm *VirtualMachine) (string, error) {
	return vb.control(vm, "pause")
}

func (vb *VBox) Resume(vm *VirtualMachine) (string, error) {
	return vb.control(vm, "resume")
}

func (vb *VBox) Reset(vm *VirtualMachine) (string, error) {
	return vb.control(vm, "reset")
}

func (vb *VBox) EnableIOAPIC(vm *VirtualMachine) (string, error) {
	return vb.modify(vm, "--ioapic", "on")
}

func (vb *VBox) EnableEFI(vm *VirtualMachine) (string, error) {
	return vb.modify(vm, "--firmware", "efi")
}

func (vb *VBox) VMInfo(uuidOrVmName string) (machine *VirtualMachine, err error) {
	out, err := vb.manage("showvminfo", uuidOrVmName, "--machinereadable")

	// lets populate the map from output strings
	m := map[string]interface{}{}
	_ = parseKeyValues(out, reKeyEqVal, func(key, val string) error {
		if strings.HasPrefix(key, "\"") {
			if k, err := strconv.Unquote(key); err == nil {
				key = k
			} //else ignore; might need to warn in log
		}
		if strings.HasPrefix(val, "\"") {
			if val, err := strconv.Unquote(val); err == nil {
				m[key] = val
			}
		} else if i, err := strconv.Atoi(val); err == nil {
			m[key] = i
		} else { // we dont expect any actually
			glog.V(6).Infof("ignoring parsing val %s for key %s", val, key)
		}
		return nil
	})

	vm := &VirtualMachine{}

	vm.UUID = m["UUID"].(string)
	vm.Spec.Name = m["name"].(string)
	path := m["CfgFile"].(string)
	if vpath, err := filepath.Rel(vb.Config.BasePath, path); err == nil {
		elems := strings.Split(vpath, string(filepath.Separator))
		if len(elems) >= 3 { //we assume the first one to be group
			vm.Spec.Group = "/" + elems[0]
		}
	}
	if path != vb.getVMSettingsFile(vm) {
		return nil, fmt.Errorf("path %s does not match expected structure", path)
	}

	vm.Spec.CPU.Count = m["cpus"].(int)
	vm.Spec.Memory.SizeMB = m["memory"].(int)

	// fill in storage details
	vm.Spec.StorageControllers = make([]StorageController, 0, 2)

	for i := 0; i < 20; i++ { // upto a 20 storage controller? :)
		sk := fmt.Sprintf("storagecontrollername%d", i)
		if v, ok := m[sk]; ok { //  e.g of v is SATA1

			sc := StorageController{Name: v.(string)}

			switch fmt.Sprintf("storagecontrollertype%d", i) {
			case string(SATA):
				sc.Type = SATA
			case string(IDE):
				sc.Type = IDE
			case string(SCSCI):
				sc.Type = SCSCI
			case string(NVME):
				sc.Type = NVME
			}

			var err error

			if si, ok := m[fmt.Sprintf("storagecontrollerinstance%d", i)]; ok {
				if sc.Instance, err = strconv.Atoi(si.(string)); err != nil {
					return nil, fmt.Errorf("wrong val")
				}
			}

			if sb, ok := m[fmt.Sprintf("storagecontrollerbootable%d", i)]; ok {
				if sc.Bootable, ok = sb.(string); !ok {
					return nil, fmt.Errorf("wrong val for storagecontrollerbootable")
				}
			}

			if sp, ok := m[fmt.Sprintf("storagecontrollerportcount%d", i)]; ok {
				if sc.PortCount, err = strconv.Atoi(sp.(string)); err != nil {
					return nil, fmt.Errorf("wrong val for storageportcount")
				}
			}

			vm.Spec.Disks = make([]Disk, 0, 2)

			for j := 0; j < sc.PortCount; j++ {
				dp := fmt.Sprintf("%s-%d-%d", v, j, 0) // key to path of disk, e.g SATA1-0-0
				if dpv, ok := m[dp]; ok && dpv != "none" {
					d := Disk{
						Path: dpv.(string),
						Controller: StorageControllerAttachment{
							Type: sc.Type,
							Port: j,
							Name: sc.Name,
						},
					}
					if duv, ok := m[fmt.Sprintf("%s-ImageUUID-%d-%d", v, j, 0)]; ok { // e.g SATA1-ImageUUID-0-0
						d.UUID = duv.(string)
					}
					vm.Spec.Disks = append(vm.Spec.Disks, d)
				}
			}
			vm.Spec.StorageControllers = append(vm.Spec.StorageControllers, sc)

		} else { //storage controllers index not found, dont loop anymore
			break
		}
	}

	// now populate network

	for i := 1; i < 20; i++ { // upto a 20 nics
		n := fmt.Sprintf("nic%d", i)

		nic := NIC{}
		if v, ok := m[n]; ok {
			if v == "none" {
				continue
			}
			nic.Mode = NetworkMode(v.(string))
		} else {
			continue
		}

		n = fmt.Sprintf("nictype%d", i)
		if v, ok := m[n]; ok {
			nic.Type = NICType(v.(string))
		}

		n = fmt.Sprintf("nicspeed%d", i)
		if v, ok := m[n]; ok {
			nic.Speedkbps, err = strconv.Atoi(v.(string))
		}

		n = fmt.Sprintf("macaddress%d", i)
		if v, ok := m[n]; ok {
			nic.MAC = v.(string)
		}

		n = fmt.Sprintf("cableconnected%d", i)
		if v, ok := m[n]; ok {
			nic.CableConnected = v.(string) == "on"
		}

		switch nic.Mode {
		case NWMode_hostonly:
			n = fmt.Sprintf("hostonlyadapter%d", i)
			if v, ok := m[n]; ok {
				nic.NetworkName = v.(string)
			}
		case NWMode_natnetwork:
			n = fmt.Sprintf("natnet%d", i)
			if v, ok := m[n]; ok {
				nic.NetworkName = v.(string)
			}
		}

		vm.Spec.NICs = append(vm.Spec.NICs, nic)
	}

	return vm, nil
}

func (vb *VBox) Define(context context.Context, vm *VirtualMachine) (*VirtualMachine, error) {

	if err := vb.EnsureVMHostPath(vm); err != nil {
		return nil, err
	}

	for i := range vm.Spec.Disks {
		disk, err := vb.EnsureDisk(context, &vm.Spec.Disks[i])
		if err != nil {
			return nil, err
		} else {
			vm.Spec.Disks[i].UUID = disk.UUID
		}

		if err != nil {
			return nil, OperationError{Path: fmt.Sprintf("disk/%d", i), Op: "ensure", Err: err}
		}
	}

	if err := vb.CreateVM(vm); err != nil && !IsAlreadyExistsError(err) {
		return nil, OperationError{Path: "vm", Op: "ensure", Err: err}
	}

	if err := vb.RegisterVM(vm); err != nil {
		return nil, OperationError{Path: "vm", Op: "ensure", Err: err}
	}

	if err := vb.SetCPUCount(vm, vm.Spec.CPU.Count); err != nil {
		return nil, OperationError{Path: "vm/cpu", Op: "set", Err: err}
	}

	if err := vb.SetMemory(vm, vm.Spec.Memory.SizeMB); err != nil {
		return nil, OperationError{Path: "vm/memory", Op: "set", Err: err}
	}

	for i, ctr := range vm.Spec.StorageControllers {
		if err := vb.AddStorageController(vm, ctr); err != nil && !IsAlreadyExistsError(err) {
			return nil, OperationError{Path: fmt.Sprintf("storagecontroller/%d", i), Op: "add", Err: err}
		}
	}

	disks := vm.Spec.Disks
	for i := range disks {
		if err := vb.AttachStorage(vm, &disks[i]); err != nil && !IsAlreadyExistsError(err) {
			return nil, OperationError{Path: fmt.Sprintf("storagecontroller/%d", i), Op: "attach", Err: err}
		}
	}

	if _, err := vb.EnableIOAPIC(vm); err != nil {
		return nil, OperationError{Path: "ioapic", Op: "enable", Err: err}
	}

	var nics = vm.Spec.NICs
	for i := range nics {
		if err := vb.AddNic(vm, &nics[i]); err != nil {
			return nil, fmt.Errorf("cannot add nic %#v", nics)
		}
	}

	if len(vm.Spec.Boot) > 0 {
		vb.SetBootOrder(vm, vm.Spec.Boot)
	}

	dvm, err := vb.VMInfo(vm.UUIDOrName())
	if err != nil || dvm.UUID == "" {
		return nil, err // to retry?
	}

	if vm.UUID == "" {
		vm.UUID = dvm.UUID
	}

	return dvm, nil
}

func (vb *VBox) EnsureVMHostPath(vm *VirtualMachine) error {
	path := vb.getVMBaseDir(vm)
	return os.MkdirAll(path, os.ModePerm)
}

// EnsureDefaults expands the vm structure to fill in details needed based on well defined conventions
// The returned instance has all the modifications and may be the same as the passed in instance
func (vb *VBox) EnsureDefaults(vm *VirtualMachine) (machine *VirtualMachine, err error) {

	verr := ValidationErrors{}
	tsctl := map[string]*StorageController{}

	for i, c := range vm.Spec.StorageControllers {
		if c.Name == "" {
			c.Name = fmt.Sprintf("%s%d", string(c.Type), i+1) // for e.g ide1
		}
		if _, ok := tsctl[c.Name]; !ok {
			tsctl[c.Name] = &c
		} else {
			verr.Add(fmt.Sprintf("storagecontroller/[%d]/", i), fmt.Errorf("duplicate name"))
		}
	}

	disks := vm.Spec.Disks

	// First ensure that we set default Buses for the disks and the ref names
	for i := range disks {
		disk := &vm.Spec.Disks[i]

		if !filepath.IsAbs(disks[i].Path) {
			disks[i].Path = fmt.Sprintf("%s/%s", vb.getVMBaseDir(vm), disks[i].Path)
		}

		if disks[i].Type == "" {
			disks[i].Type = HDDrive
		}

		if disk.Controller.Type == "" {
			disk.Controller.Type = SATA
		}

		if disk.Controller.Name == "" {
			sctlName := fmt.Sprintf("%s1", string(disk.Controller.Type))
			disk.Controller.Name = sctlName // default to 1, for e.g ide1
			// auto create a storage controller if one does not already exist
			if _, ok := tsctl[sctlName]; !ok {
				tsctl[sctlName] = &StorageController{Name: sctlName, Type: disk.Controller.Type}
			}
		}

		if disks[i].Format == "" {
			disks[i].Format = VDI
		}
	}

	// counts track the number of disks using a storage controller
	counts := map[string]int{}

	// now set back the storage controllers to VM
	vm.Spec.StorageControllers = []StorageController{}
	for k, v := range tsctl {
		vm.Spec.StorageControllers = append(vm.Spec.StorageControllers, *v)
		counts[k] = 0
	}

	// now ensure that we account for all user set and auto assigned (defaulted) value and attach them to ports
	for i := range disks {

		if disks[i].Path == "" {
			verr.Add(fmt.Sprintf("disk/%d", i), fmt.Errorf("disk path is empty, needs an absolute file path"))
		}

		if count, ok := counts[disks[i].Controller.Name]; ok {
			counts[disks[i].Controller.Name] = count + 1
			switch disks[i].Controller.Type {
			case IDE:
				disks[i].Controller.Port = count / 2
				disks[i].Controller.Device = count % 2
			case SATA:
				disks[i].Controller.Port = count
			default:
				disks[i].Controller.Port = count
				glog.Warning("trying to default the port for controller type %s, this might not work", disks[i].Controller.Type)
			}

		} else {
			verr.Add(fmt.Sprintf("disk/%d", i), fmt.Errorf("storagecontroller ref %s did not resolve", disks[i].Controller.Name))
		}
	}

	if err := vb.SetNICDefaults(vm); err != nil {
		return nil, err
	}

	if len(verr.errors) > 0 {
		return nil, verr
	} else { // return the same instance
		return vm, nil
	}
}

func isAlreadyExistErrorMessage(out string) bool {
	return strings.Contains(out, "already exists")
}
