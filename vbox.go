package virtualbox

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/glog"
)

const (
	VBoxManage  = "VBoxManage"
	NatPortBase = 10900
)

var DefaultVBBasePath = GetDefaultVBBasePath()

var reColonLine = regexp.MustCompile(`([^:]+):\s+(.*)`)

// parses lines like the following
//  foo="bar"
var reKeyEqVal = regexp.MustCompile(`([^=]+)=\s*(.*)`)

// Config for the Manager
type Config struct {
	// BasePath is the base filesystem location for managing this provider's configuration
	// Defaults to $HOME/.vbm/VBox BasePath string
	BasePath string
	// VirtualBoxPath is where the VirtualBox cmd is available on the local machine
	VirtualBoxPath string

	Groups []string

	// expected to be managed by this tool
	Networks []Network
}

// VBox uses the VBoxManage command for its functionality
type VBox struct {
	Config  Config
	Verbose bool
	// as discovered and includes networks created out of band (not through this api)
	// TODO: Merge them to a single map and provide accessors for specific filtering
	HostOnlyNws map[string]*Network
	BridgedNws  map[string]*Network
	InternalNws map[string]*Network
	NatNws      map[string]*Network
}

func NewVBox(config Config) *VBox {
	if config.BasePath == "" {
		config.BasePath = DefaultVBBasePath
	}
	return &VBox{
		Config:      config,
		HostOnlyNws: make(map[string]*Network),
		BridgedNws:  make(map[string]*Network),
		InternalNws: make(map[string]*Network),
		NatNws:      make(map[string]*Network),
	}
}

func GetDefaultVBBasePath() string {
	user, err := user.Current()
	if err != nil {
		panic(fmt.Errorf("basepath not supplied and default location cannot be determined %v", err))
	}
	return fmt.Sprintf("%s/VirtualBox VMs", user.HomeDir)
}

func IsVBoxError(err error) bool {
	_, ok := err.(VBoxError)
	return ok
}

//VBoxError are errors that are returned as error by Virtualbox cli on stderr
type VBoxError string

func (ve VBoxError) Error() string {
	return string(ve)
}

func (vb *VBox) getVMBaseDir(vm *VirtualMachine) string {
	var group string
	if vm.Spec.Group != "" {
		group = vm.Spec.Group
	}

	return filepath.Join(vb.Config.BasePath, group, vm.Spec.Name)
}

func (vb *VBox) getVMSettingsFile(vm *VirtualMachine) string {
	return filepath.Join(vb.getVMBaseDir(vm), vm.Spec.Name+".vbox")
}

func (vb *VBox) manage(args ...string) (string, error) {
	vboxManage := vboxManagePath()
	cmd := exec.Command(vboxManage, args...)
	glog.V(4).Infof("COMMAND: %v %v", vboxManage, strings.Join(args, " "))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	stderrStr := stderr.String()

	if err != nil {
		if ee, ok := err.(*exec.Error); ok && ee.Err == exec.ErrNotFound {
			return "", errors.New("unable to find VBoxManage command in path")
		}
		return "", VBoxError(stderrStr)
	}

	glog.V(10).Infof("STDOUT:\n{\n%v}", stdout.String())
	glog.V(10).Infof("STDERR:\n{\n%v}", stderrStr)

	return string(stdout.Bytes()), err
}

func (vb *VBox) modify(vm *VirtualMachine, args ...string) (string, error) {
	return vb.manage(append([]string{"modifyvm", vm.UUIDOrName()}, args...)...)
}

func (vb *VBox) control(vm *VirtualMachine, args ...string) (string, error) {
	return vb.manage(append([]string{"controlvm", vm.UUIDOrName()}, args...)...)
}

func (vb *VBox) ListDHCPServers() (map[string]*DHCPServer, error) {
	listOutput, err := vb.manage("list", "dhcpservers")
	if err != nil {
		return nil, err
	}

	m := make(map[string]*DHCPServer)

	var dhcpServer *DHCPServer

	err = parseKeyValues(listOutput, reColonLine, func(key, val string) error {
		switch key {
		case "NetworkName":
			dhcpServer = &DHCPServer{}
			m[val] = dhcpServer
			dhcpServer.NetworkName = val
		case "IP":
			dhcpServer.IPAddress = val
		case "upperIPAddress":
			dhcpServer.UpperIPAddress = val
		case "lowerIPAddress":
			dhcpServer.LowerIPAddress = val
		case "NetworkMask":
			dhcpServer.NetworkMask = val
		case "Enabled":
			dhcpServer.Enabled = val == "Yes"
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (vb *VBox) ListOSTypes() (map[string]*OSType, error) {
	listOutput, err := vb.manage("list", "ostypes")
	if err != nil {
		return nil, err
	}

	m := make(map[string]*OSType)

	var osType *OSType

	err = parseKeyValues(listOutput, reColonLine, func(key, val string) error {
		switch key {
		case "ID":
			osType = &OSType{}
			m[val] = osType
			osType.ID = val
		case "Description":
			osType.Description = val
		case "Family ID":
			osType.FamilyID = val
		case "Family Desc":
			osType.FamilyDescription = val
		case "64 bit":
			osType.Bit64 = val == "true"
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (vb *VBox) MarkHDImmutable(hdPath string) error {
	vb.manage("modifyhd", hdPath, "--type", "immutable")
	return nil
}
