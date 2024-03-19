package virtualbox

import "net"

type StorageControllerType string

const (
	IDE   = StorageControllerType("IDE")
	SATA  = StorageControllerType("SATA")
	SCSCI = StorageControllerType("SCSCI")
	NVME  = StorageControllerType("NVME")
)

type DiskType string

const (
	DVDDrive = DiskType("dvddrive")
	HDDrive  = DiskType("hdd")
	FDDrive  = DiskType("fdd")
)

func (d DiskType) ForShowMedium() string {
	switch d {
	case DVDDrive:
		return "dvd"
	case HDDrive:
		return "disk"
	case FDDrive:
		return "floppy"
	}
	return ""
}

type VirtualMachineState string

const (
	Poweroff = VirtualMachineState("poweroff")
	Running  = VirtualMachineState("running")
	Paused   = VirtualMachineState("paused")
	Saved    = VirtualMachineState("saved")
	Aborted  = VirtualMachineState("aborted")
)

type Disk struct {
	// Path represents the absolute path in the system where the disk is stored, normally is under the vm folder
	Path          string
	SizeMB        int64
	Format        DiskFormat
	UUID          string
	Controller    StorageControllerAttachment
	Type          DiskType
	NonRotational bool
	AutoDiscard   bool
}

type StorageControllerAttachment struct {
	// Type represents the storage controller, rest of the fields needs to interpreted based on this
	Type   StorageControllerType
	Port   int
	Device int
	// Name of the storage controller target for this attachment
	Name string
}

type StorageController struct {
	Name      string
	Type      StorageControllerType
	Instance  int
	PortCount int
	Bootable  string //on, off
}

type Snapshot struct {
	Name        string
	Description string
}

type CPU struct {
	Count int
}

type Memory struct {
	SizeMB int
}

type NetworkMode string

const (
	NWMode_none       = NetworkMode("none")
	NWMode_null       = NetworkMode("null")
	NWMode_nat        = NetworkMode("nat")
	NWMode_natnetwork = NetworkMode("natnetwork")
	NWMode_bridged    = NetworkMode("bridged")
	NWMode_intnet     = NetworkMode("intnet")
	NWMode_hostonly   = NetworkMode("hostonly")
	NWMode_generic    = NetworkMode("generic")
)

type NICType string

const (
	NIC_Am79C970A = NICType("Am79C970A")
	NIC_Am79C973  = NICType("Am79C973")
	NIC_82540EM   = NICType("82540EM")
	NIC_82543GC   = NICType("82543GC")
	NIC_82545EM   = NICType("82545EM")
	NIC_virtio    = NICType("virtio")
)

type NIC struct {
	Index int
	//	Name            string      // device name of this nic, used for correlation with Adapters
	Mode            NetworkMode // nat, hostonly etc
	NetworkName     string      //optional name of the Network to connect this nic to. For hostnetwork and int this is the same as the host device
	Type            NICType
	CableConnected  bool
	Speedkbps       int
	BootPrio        int
	PromiscuousMode string
	MAC             string //auto assigns mac automatically
	PortForwarding  []PortForwarding
}

type NetProtocol string

const (
	TCP = NetProtocol("tcp")
	UDP = NetProtocol("udp")
)

type PortForwarding struct {
	Index     int
	Name      string
	Protocol  NetProtocol
	HostIP    string
	HostPort  int
	GuestIP   string
	GuestPort int
}

type Network struct {
	GUID       string
	Name       string
	IPNet      net.IPNet
	Mode       NetworkMode
	DeviceName string
	HWAddress  string
}

type BootDevice string

const BOOT_net, BOOT_disk, BOOT_none BootDevice = "net", "disk", "none"

type VirtualMachineSpec struct {
	// Name identifies the vm and is also used in forming full path, see VBox.BasePath
	Name               string
	Group              string
	Disks              []Disk
	CPU                CPU
	Memory             Memory
	NICs               []NIC
	OSType             OSType
	StorageControllers []StorageController
	Boot               []BootDevice
	State              VirtualMachineState
	Snapshots          []Snapshot
	CurrentSnapshot    Snapshot
	DragAndDrop        string
	Clipboard          string
}

type VirtualMachine struct {
	UUID string
	Spec VirtualMachineSpec
}

func (vm *VirtualMachine) UUIDOrName() string {
	if vm.UUID == "" {
		return vm.Spec.Name
	} else {
		return vm.UUID
	}
}

type DHCPServer struct {
	IPAddress      string
	NetworkName    string
	NetworkMask    string
	LowerIPAddress string
	UpperIPAddress string
	Enabled        bool
}

type OSType struct {
	ID                string
	Description       string
	FamilyID          string
	FamilyDescription string
	Bit64             bool
}
