package virtualbox

import (
	"context"
	"fmt"
	diff "gopkg.in/d4l3k/messagediff.v1"
	"testing"
	"time"
)

func TestVBox_Netinfo(t *testing.T) {
	verifyNetwork := func(mode string, nws []Network) {

		for i := range nws {
			nw := nws[i]

			if nw.Name == "" {
				t.Errorf("does not have name set for modetype %s, got %#v", mode, nw)
			}
			if nw.Mode == "" {
				t.Errorf("does not have mode set for modetype %s, got %#v", mode, nw)
			}
			if nw.Mode == NWMode_hostonly || nw.Mode == NWMode_bridged {
				if nw.DeviceName == "" {
					t.Errorf("does not have devicename")
				}
			}
		}
	}

	vb := NewVBox(Config{})
	if nws, err := vb.HostOnlyNetInfo(); err != nil {
		t.Errorf("error %#v", err)
	} else {
		verifyNetwork("hostonly", nws)
	}

	if nws, err := vb.NatNetInfo(); err != nil {
		t.Errorf("error %#v", err)
	} else {
		verifyNetwork("nat", nws)
	}

	if nws, err := vb.BridgeNetInfo(); err != nil {
		t.Errorf("error %#v", err)
	} else {
		verifyNetwork("bridge", nws)
	}

	if nws, err := vb.InternalNetInfo(); err != nil {
		t.Errorf("error %#v", err)
	} else {
		verifyNetwork("internal", nws)
	}
}

func TestSetNetDefaults(t *testing.T) {
	// Object under test
	vb := NewVBox(Config{})

	nic1 := NIC{}
	nic2 := NIC{}

	vm := &VirtualMachine{}
	vm.Spec.Name = "testvm1"
	vm.Spec.Group = "/tess"
	vm.Spec.OSType = Linux64
	vm.Spec.CPU.Count = 2
	vm.Spec.Memory.SizeMB = 1000
	vm.Spec.NICs = []NIC{nic1, nic2}

	// Method under test
	vb.SetNICDefaults(vm)

	if len(vm.Spec.NICs) == 0 {
		fmt.Errorf("expected nics, got none")
	}

	for i := range vm.Spec.NICs {
		nic := &vm.Spec.NICs[i]
		if nic.Mode != NWMode_hostonly {
			t.Errorf("expected %s, got %s", NWMode_hostonly, nic.Mode)
		}

		if nic.NetworkName == "" {
			t.Errorf("expected a non nil networkname")
		}
	}

	fmt.Printf("%#v", vm.Spec.NICs)
}

func TestSyncetwork(t *testing.T) {
	vb := NewVBox(Config{})

	network := &Network{Mode: NWMode_hostonly}
	err := vb.CreateNet(network)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	defer vb.DeleteNet(network)

	if network.Name == "" {
		t.Errorf("expected name")
	}

	err = vb.SyncNICs()
	if err != nil {
		t.Fatalf("error syncing %#v", err)
	}

	//alteast we should find the network we created
	if nw1, ok := vb.HostOnlyNws[network.Name]; !ok {
		t.Fatalf("error syncing %#v", err)
	} else {
		if network.Name != nw1.Name {
			t.Errorf("Not the same name, got %s", nw1.Name)
		}
	}
}

func TestVBox_Ensure(t *testing.T) {
	// Object under test
	vb := NewVBox(Config{})

	nic1 := NIC{
		Mode:        NWMode_hostonly,
		NetworkName: "vboxnet0",
		BootPrio:    0,
	}
	nic2 := NIC{
		Mode:        NWMode_intnet,
		NetworkName: "intnet0",
		BootPrio:    0,
	}

	vm := &VirtualMachine{}
	vm.Spec.Name = "testvm1"
	vm.Spec.Group = "/tess"
	vm.Spec.OSType = Linux64
	vm.Spec.CPU.Count = 2
	vm.Spec.Memory.SizeMB = 1000
	vm.Spec.NICs = []NIC{nic1, nic2}

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
		t.Errorf("Error defining %#v", err)
	} else {
		t.Logf("Defined vm %#v", nvm)
	}

	fmt.Printf("Created %#v\n", vm)
	fmt.Printf("Created %#v\n", nvm)

	if diff, equal := diff.PrettyDiff(vm, nvm); !equal {
		t.Logf("%s", diff) // we need to fix these diffs
	}
}
