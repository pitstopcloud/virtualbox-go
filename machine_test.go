package virtualbox

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/golang/glog"
	diff "gopkg.in/d4l3k/messagediff.v1"
)

func init() {
	flag.Set("logtostderr", "true")
	flag.Set("v", "10")
}

func TestVBox_Define(t *testing.T) {

	// Object under test
	vb := NewVBox(Config{})

	disk1 := Disk{
		Path:   "disk1.vdi",
		SizeMB: 10,
	}

	vm := &VirtualMachine{}
	vm.Spec.Name = "vm01"
	vm.Spec.Group = "/example"
	vm.Spec.OSType = Linux64
	vm.Spec.CPU.Count = 2
	vm.Spec.Memory.SizeMB = 1000
	vm.Spec.Disks = []Disk{disk1}

	vb.EnsureDefaults(vm)

	ctx := context.Background()
	context.WithTimeout(ctx, 1*time.Minute)

	vb.UnRegisterVM(vm)
	vb.DeleteVM(vm)

	defer vb.DeleteVM(vm)
	defer vb.UnRegisterVM(vm)

	nvm, err := vb.Define(ctx, vm)
	if err != nil {
		t.Errorf("Error %+v", err)
	}

	fmt.Printf("Created %#v\n", vm)
	fmt.Printf("Created %#v\n", nvm)

	if diff, equal := diff.PrettyDiff(vm, nvm); !equal {
		t.Logf("%s", diff) // we need to fix these diffs
	}

}

func TestVBox_SetStates(t *testing.T) {

	// Object under test
	vb := NewVBox(Config{})

	disk1 := Disk{
		Path:   "disk1.vdi",
		SizeMB: 10,
	}

	vm := &VirtualMachine{}
	vm.Spec.Name = "vm01"
	vm.Spec.Group = "/example"
	vm.Spec.OSType = Linux64
	vm.Spec.CPU.Count = 2
	vm.Spec.Memory.SizeMB = 1000
	vm.Spec.Disks = []Disk{disk1}

	// Method under test
	vb.EnsureDefaults(vm)

	ctx := context.Background()
	context.WithTimeout(ctx, 1*time.Minute)

	vb.UnRegisterVM(vm)
	vb.DeleteVM(vm)

	//defer vb.DeleteVM(vm)
	//defer vb.UnRegisterVM(vm)

	nvm, err := vb.Define(ctx, vm)

	if err != nil {
		t.Fatalf("%v", err)
	} else if nvm.UUID == "" {
		t.Fatalf("VM not discvoerable after creation %s", vm.Spec.Name)
	}

	_, err = vb.Start(vm)
	if err != nil {
		t.Fatalf("Failed to start vm %s, error %v", vm.Spec.Name, err)
	}

	_, err = vb.Stop(vm)
	if err != nil {
		t.Fatalf("Failed to stop vm %s, error %v", vm.Spec.Name, err)
	}

	_ = nvm
	_ = err
}

func TestVBox_EnsureDefaults(t *testing.T) {

	// Object under test
	vb := NewVBox(Config{})

	disk1 := Disk{
		Path: "disk1.vdi",
	}

	disk2 := Disk{
		Path:   "disk2.vdi",
		SizeMB: 10,
		Controller: StorageControllerAttachment{
			Type: NVME,
		},
	}

	disk3 := Disk{
		Path:   "disk3.vdi",
		SizeMB: 10,
		Controller: StorageControllerAttachment{
			Type: IDE,
		},
	}

	disk4 := Disk{
		Path:   "disk4.vdi",
		SizeMB: 10,
		Controller: StorageControllerAttachment{
			Type: SCSCI,
		},
	}

	vm := &VirtualMachine{}
	vm.Spec.Name = "testvm1"
	vm.Spec.OSType = Linux64
	vm.Spec.CPU.Count = 2
	vm.Spec.Memory.SizeMB = 1000
	vm.Spec.Disks = []Disk{disk1, disk2, disk3, disk4}

	// Method under test
	vb.EnsureDefaults(vm)

	if len(vm.Spec.StorageControllers) != 4 {
		t.Errorf("Expected stroage cotnroller to be auto created")
	}

	sort.Slice(vm.Spec.StorageControllers, func(i, j int) bool {
		return vm.Spec.StorageControllers[i].Name < vm.Spec.StorageControllers[j].Name
	})

	if vm.Spec.StorageControllers[0].Type != IDE ||
		vm.Spec.StorageControllers[1].Type != NVME ||
		vm.Spec.StorageControllers[2].Type != SATA ||
		vm.Spec.StorageControllers[3].Type != SCSCI {
		t.Errorf("Expected 4 storage controller to be auto created, got %+v", vm.Spec.StorageControllers)
	}

	if vm.Spec.Disks[0].Controller.Type != SATA {
		t.Errorf("Expected disks type to be SATA, got %v", vm.Spec.Disks[0].Type)
	}

	for i := range vm.Spec.Disks {
		if !filepath.IsAbs(vm.Spec.Disks[i].Path) {
			t.Errorf("Expected disks path to be absolute, relative to vm path, got %v", vm.Spec.Disks[i].Path)
		}

		if vm.Spec.Disks[i].Type != HDDrive {
			t.Errorf("Expected disks type to be default set to HDD got %v", vm.Spec.Disks[i].Type)
		}
	}

}

func TestVBox_CreateVM(t *testing.T) {
	glog.V(10).Info("setup")

	dirName, err := ioutil.TempDir("", "vbm")
	if err != nil {
		t.Errorf("Tempdir creation failed %v", err)
	}
	defer os.RemoveAll(dirName)

	vb := NewVBox(Config{
		BasePath: dirName,
	})

	disk1 := Disk{
		Path:   filepath.Join(dirName, "disk1.vdi"),
		Format: VDI,
		SizeMB: 10,
	}

	err = vb.CreateDisk(&disk1)
	if err != nil {
		t.Errorf("CreateDisk failed %v", err)
	}

	vm := &VirtualMachine{}
	vm.Spec.Name = "testvm1"
	vm.Spec.OSType = Linux64
	vm.Spec.CPU.Count = 2
	vm.Spec.Memory.SizeMB = 1000
	vm.Spec.Disks = []Disk{disk1}

	err = vb.CreateVM(vm)
	if err != nil {
		t.Fatalf("Failed creating vm %v", err)
	}

	err = vb.RegisterVM(vm)
	if err != nil {
		t.Fatalf("Failed registering vm")
	}
}

func TestVBox_CreateVMDefaultPath(t *testing.T) {
	glog.V(10).Info("setup")

	// No BasePath specified
	vb := NewVBox(Config{})

	vm := &VirtualMachine{}
	vm.Spec.Name = "testvm1"
	vm.Spec.OSType = Linux64
	vm.Spec.CPU.Count = 2
	vm.Spec.Memory.SizeMB = 1000

	err := vb.CreateVM(vm)
	if err != nil {
		t.Fatalf("Failed creating vm %v", err)
	}

	err = vb.RegisterVM(vm)
	if err != nil {
		t.Fatalf("Failed registering vm")
	}

	err = vb.UnRegisterVM(vm)
	if err != nil {
		t.Fatalf("Failed registering vm")
	}

	vb.DeleteVM(vm)
	if err != nil {
		t.Fatalf("Failed registering vm")
	}
}

var showVmInfoOutput = `
name="testvm1"
groups="/tess,/tess2"
ostype="Other Linux (32-bit)"
UUID="6aa44e71-71c6-4e68-a61f-f69e133ecffa"
CfgFile="/Users/araveendrann/VirtualBox VMs/tess/testvm1/testvm1.vbox"
SnapFldr="/Users/araveendrann/VirtualBox VMs/tess/testvm1/Snapshots"
LogFldr="/Users/araveendrann/VirtualBox VMs/tess/testvm1/Logs"
hardwareuuid="6aa44e71-71c6-4e68-a61f-f69e133ecffa"
memory=128
pagefusion="off"
vram=8
cpuexecutioncap=100
hpet="off"
chipset="piix3"
firmware="BIOS"
cpus=1
pae="on"
longmode="off"
triplefaultreset="off"
apic="on"
x2apic="on"
cpuid-portability-level=0
bootmenu="messageandmenu"
boot1="floppy"
boot2="dvd"
boot3="disk"
boot4="none"
acpi="on"
ioapic="off"
biosapic="apic"
biossystemtimeoffset=0
rtcuseutc="off"
hwvirtex="on"
nestedpaging="on"
largepages="on"
vtxvpid="on"
vtxux="on"
paravirtprovider="default"
effparavirtprovider="kvm"
VMState="poweroff"
VMStateChangeTime="2017-12-10T01:18:02.000000000"
monitorcount=1
accelerate3d="off"
accelerate2dvideo="off"
teleporterenabled="off"
teleporterport=0
teleporteraddress=""
teleporterpassword=""
tracing-enabled="off"
tracing-allow-vm-access="off"
tracing-config=""
autostart-enabled="off"
autostart-delay=0
defaultfrontend=""
storagecontrollername0="SATA1"
storagecontrollertype0="IntelAhci"
storagecontrollerinstance0="0"
storagecontrollermaxportcount0="30"
storagecontrollerportcount0="30"
storagecontrollerbootable0="on"
"SATA1-0-0"="/Users/araveendrann/VirtualBox VMs/tess/testvm1/disk1.vdi"
"SATA1-ImageUUID-0-0"="38f0cf9d-6c60-4f59-ba0b-cd1dfb5329d6"
"SATA1-1-0"="none"
"SATA1-2-0"="none"
"SATA1-3-0"="none"
"SATA1-4-0"="none"
"SATA1-5-0"="none"
"SATA1-6-0"="none"
"SATA1-7-0"="none"
"SATA1-8-0"="none"
"SATA1-9-0"="none"
"SATA1-10-0"="none"
"SATA1-11-0"="none"
"SATA1-12-0"="none"
"SATA1-13-0"="none"
"SATA1-14-0"="none"
"SATA1-15-0"="none"
"SATA1-16-0"="none"
"SATA1-17-0"="none"
"SATA1-18-0"="none"
"SATA1-19-0"="none"
"SATA1-20-0"="none"
"SATA1-21-0"="none"
"SATA1-22-0"="none"
"SATA1-23-0"="none"
"SATA1-24-0"="none"
"SATA1-25-0"="none"
"SATA1-26-0"="none"
"SATA1-27-0"="none"
"SATA1-28-0"="none"
"SATA1-29-0"="none"
natnet1="nat"
macaddress1="080027220665"
cableconnected1="on"
nic1="nat"
nictype1="Am79C973"
nicspeed1="0"
mtu="0"
sockSnd="64"
sockRcv="64"
tcpWndSnd="64"
tcpWndRcv="64"
nic2="none"
nic3="none"
nic4="none"
nic5="none"
nic6="none"
nic7="none"
nic8="none"
hidpointing="ps2mouse"
hidkeyboard="ps2kbd"
uart1="off"
uart2="off"
uart3="off"
uart4="off"
lpt1="off"
lpt2="off"
audio="coreaudio"
clipboard="disabled"
draganddrop="disabled"
vrde="off"
usb="off"
ehci="off"
xhci="off"
vcpenabled="off"
vcpscreens=0
vcpfile="/Users/araveendrann/VirtualBox VMs/tess/testvm1/testvm1.webm"
vcpwidth=1024
vcpheight=768
vcprate=512
vcpfps=25
GuestMemoryBalloon=0
`
