package virtualbox

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"testing"
)

func TestVbox_CreateDelete(t *testing.T) {

	dirName, err := ioutil.TempDir("", "vbm")
	if err != nil {
		t.Errorf("CreateDisk failed %v", err)
	}
	defer os.RemoveAll(dirName)

	expected := Disk{
		Path:   filepath.Join(dirName, "disk1.vdi"),
		Format: VDI,
		SizeMB: 10,
	}

	var vb VBox

	err = vb.CreateDisk(&expected)
	if err != nil {
		t.Errorf("CreateDisk failed %v", err)
	}

	actual, err := vb.DiskInfo(&expected)
	if err != nil {
		t.Fatalf("DiksInfo failed with %v", err)
	}

	if actual.UUID == "" {
		t.Fatalf("Disk was not created?")
	}

	err = vb.DeleteDisk(actual.UUID)
	if err != nil {
		t.Fatalf("error deleting disk %v", err)
	}

	actual, err = vb.DiskInfo(&expected)
	if err == nil {
		t.Fatalf("Expected error, but gone none")
	}

	if !IsDiskNotFound(err) {
		t.Fatalf("Expected DiskNotFoundError error, but gone %v", err)
	}

}

func TestShowMediumOutputRegex(t *testing.T) {
	var sampleDiskOut = `
UUID:           0e3f0c1b-f523-4a50-b1a8-d1e8c9a508b4
Parent UUID:    base
State:          created
Type:           normal (base)
Storage format: VDI
Format variant: dynamic default
Capacity:       1000 MBytes
Size on disk:   2 MBytes
Encryption:     disabled
`

	user, _ := user.Current()
	expected := Disk{
		Path:   user.HomeDir + "/.vbm/Vbox/myvm1/disk1.vdi",
		Format: VDI,
		UUID:   "0e3f0c1b-f523-4a50-b1a8-d1e8c9a508b4",
	}

	//Vbox.CreateDisk(disk1.Path, )
	var disk = Disk{}
	_ = parseKeyValues(sampleDiskOut, reColonLine, func(key, val string) error {
		switch key {
		case "UUID":
			disk.UUID = val
		case "Location":
			disk.Path = val
		case "Storage format":
			disk.Format = DiskFormat(val)
		}

		return nil
	})

	if !reflect.DeepEqual(expected, disk) {
		t.Errorf("Did not parse showmediuminfo out to disk as expected. Got %+v", disk)
	}

}
