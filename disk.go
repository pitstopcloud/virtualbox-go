package virtualbox

import (
	"context"
	"fmt"
	"strings"
)

type DiskFormat string

const (
	VDI  = DiskFormat("VDI")
	VMDK = DiskFormat("VMDK")
	VHD  = DiskFormat("VHD")
)

type DiskNotFoundError string

func (d DiskNotFoundError) Error() string {
	return string(d)
}

func IsDiskNotFound(err error) bool {
	_, ok := err.(DiskNotFoundError)
	return ok
}

func (vb *VBox) EnsureDisk(context context.Context, disk *Disk) (*Disk, error) {
	d, err := vb.DiskInfo(disk)
	if IsDiskNotFound(err) {
		err = vb.CreateDisk(disk)
		if err != nil {
			return nil, err
		} else {
			d, err = vb.DiskInfo(disk)
		}
	}

	return d, err
}

func (disk *Disk) UUIDorPath() string {
	var uuidOrPath string
	if disk.UUID != "" {
		uuidOrPath = disk.UUID
	} else {
		uuidOrPath = disk.Path
	}
	return uuidOrPath
}

func (vb *VBox) DiskInfo(disk *Disk) (*Disk, error) {
	args := []string{"showmediuminfo"}
	if disk.Type != "" {
		args = append(args, disk.Type.ForShowMedium())
	}
	args = append(args, disk.UUIDorPath())
	out, err := vb.manage(args...)
	if err != nil {
		if IsVBoxError(err) && isFileNotFoundMessage(err.Error()) {
			return nil, DiskNotFoundError(out)
		}
		return nil, err
	}

	var ndisk Disk
	_ = parseKeyValues(out, reColonLine, func(key, val string) error {
		switch key {
		case "UUID":
			ndisk.UUID = val
		case "Location":
			ndisk.Path = val
		case "Storage format":
			ndisk.Format = DiskFormat(val)
		}

		return nil
	})

	if ndisk.UUID == "" {
		return &ndisk, DiskNotFoundError(disk.UUIDorPath())
	}

	return &ndisk, nil
}

func (vb *VBox) CreateDisk(disk *Disk) error {
	if disk.Format == "" {
		disk.Format = VDI
	}

	_, err := vb.manage("createmedium", "disk", "--filename", disk.Path, "--size", fmt.Sprintf("%d", disk.SizeMB),
		"--format", string(disk.Format))

	return err
}

func (vb *VBox) DeleteDisk(uuidOfFile string) error {
	out, err := vb.manage("closemedium", uuidOfFile, "--delete")
	if err != nil {
		if isFileNotFoundMessage(out) {
			return DiskNotFoundError(out)
		}
	}
	return err
}

func isFileNotFoundMessage(out string) bool {
	return strings.Contains(out, "VERR_FILE_NOT_FOUND")
}
