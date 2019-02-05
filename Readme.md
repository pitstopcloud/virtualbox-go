
# virtualbox-go
The most complete Golang wrapper for [virtualbox](https://www.virtualbox.org/) for macOS (Not tested on other platforms).  This library includes a wide range of support for different virtualbox operations that include dhcp, disk, network, virtualmachines, power, etc. You can use this library to stitch your own network and compute topology using virtualbox. Refer to examples for more details.

## What is Virtualbox ?
From [here](https://www.virtualbox.org/manual/ch01.html), Oracle VM VirtualBox is a cross-platform virtualization application. What does that mean? For one thing, it installs on your existing Intel or AMD-based computers, whether they are running Windows, Mac OS X, Linux, or Oracle Solaris operating systems (OSes). Secondly, it extends the capabilities of your existing computer so that it can run multiple OSes, inside multiple virtual machines, at the same time. As an example, you can run Windows and Linux on your Mac, run Windows Server 2016 on your Linux server, run Linux on your Windows PC, and so on, all alongside your existing applications. You can install and run as many virtual machines as you like. The only practical limits are disk space and memory.

Use cases include:

 - Testing new network topology
 - Testing new routing methodologies
 - Testing new container networking
 - Testing new Kubernetes network plugins
 - Testing new software defined firewalls and gateways
 - Local simulation of a datacenter L2/L3/L4 usecases.

## Prerequisites

 - MacOS > High Sierra 10.13.1 (Not tested on other OS)
 - Virtualbox > 5.1.28r117968
 - Golang > 1.5

## Installation
You can add virtualbox-go to your GOPATH by running:
```bash
go get -u github.com/uruddarraju/virtualbox-go
```
You can then add the following imports to your golang code for you to start using it and having fun:
```go
import (
    vbg "github.com/uruddarraju/virtualbox-go"
)
```

## Examples

### Create a Virtual Machine
```go
func CreateVM() {
    // setup temp directory, that will be used to cache different VM related files during the creation of the VM.
    dirName, err := ioutil.TempDir("", "vbm")  
    if err != nil {  
       t.Errorf("Tempdir creation failed %v", err)  
    }
    defer os.RemoveAll(dirName)  
      
    vb := vbg.NewVBox(vbg.Config{  
       BasePath: dirName,  
    })  
      
    disk1 := vbg.Disk{  
      Path:   filepath.Join(dirName, "disk1.vdi"),  
      Format: VDI,  
      SizeMB: 10,  
    }  
      
    err = vb.CreateDisk(&disk1)  
    if err != nil {  
       t.Errorf("CreateDisk failed %v", err)  
    }  
      
    vm := &vbg.VirtualMachine{}  
    vm.Spec.Name = "testvm1"  
    vm.Spec.OSType = Linux64  
    vm.Spec.CPU.Count = 2  
    vm.Spec.Memory.SizeMB = 1000  
    vm.Spec.Disks = []vbg.Disk{disk1}  
      
    err = vb.CreateVM(vm)  
    if err != nil {  
       t.Fatalf("Failed creating vm %v", err)  
    }  
      
    err = vb.RegisterVM(vm)  
    if err != nil {  
       t.Fatalf("Failed registering vm")  
    }
}
```

### Get VM Info
```go
func GetVMInfo(name string) (machine *vbm.VirtualMachine, err error) {
    vb := vbg.NewVBox(vbg.Config{})
    return vb.VMInfo(name)
}
```

### Managing states of a Virtual Machine
```go
func ManageStates(vm *vbg.VirtualMachine) {
    vb := vbg.NewVBox(vbg.Config{})
    ctx := context.Background()  
    context.WithTimeout(ctx, 1*time.Minute)
    // Start a VM, this call is idempotent.
    _, err = vb.Start(vm)  
    if err != nil {  
       t.Fatalf("Failed to start vm %s, error %v", vm.Spec.Name, err)  
    }  
    
    // Reset a VM
    _, err = vb.Reset(vm)  
    if err != nil {  
       t.Fatalf("Failed to reset vm %s, error %v", vm.Spec.Name, err)  
    }
    
    // Pause and Resume VMs
    _, err = vb.Pause(vm)  
    if err != nil {  
       t.Fatalf("Failed to pause vm %s, error %v", vm.Spec.Name, err)  
    }
    _, err = vb.Resume(vm)  
    if err != nil {  
       t.Fatalf("Failed to resume vm %s, error %v", vm.Spec.Name, err)  
    }
    
    // Stop a VM, this call is also idempotent.
    _, err = vb.Stop(vm)  
    if err != nil {  
       t.Fatalf("Failed to stop vm %s, error %v", vm.Spec.Name, err)  
    }
}
```

### Attach New Disk to existing VM
```go
func AttachDisk(vm *vbg.VirtualMachine) error {
    disk2 := &vbg.Disk{  
      Path:   filepath.Join(dirName, "disk2.vdi"),  
      Format: VDI,  
      SizeMB: 100,  
    }  
    vb := vbg.NewVBox(vbg.Config{})
    ctx := context.Background()  
    context.WithTimeout(ctx, 1*time.Minute)
    vb.AttachStorage(vm, disk2)
}
```

### More Documentation
Coming soon....  

## Why did I build it and not use something else ?
I looked around a lot for clean implementations of virtualbox golang wrappers, but could not find anything with high quality and test coverage, and also most libraries out there only have support for a few operations in virtualbox but not everything. The closest match to my requirements was [libmachine from docker](https://github.com/docker/machine/tree/master/libmachine), but it had some tight coupling with the rest of docker packages which I definitely did not want to use. Also, it was very difficult finding something that had god documentation and examples. You might not find these to be good enough reasons to start a new project, but I did found them compelling enough to start my own. 