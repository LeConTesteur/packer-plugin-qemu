package qemu

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type Disk struct {
	FullPath string
	Size     string
	Source   string
	Format   string
}

type ActionDisk struct {
	Disk        Disk
	QemuImgArgs QemuImgArgs
	Driver      Driver
	ActionType  string
}

func (d *ActionDisk) RunCommand() error {
	if d.ActionType == "copy" {
		return d.RunCopyCommand()
	}
	var command []string
	switch {
	case d.ActionType == "create":
		command = d.BuildCreateCommand()
	case d.ActionType == "convert":
		command = d.BuildConvertCommand()
	case d.ActionType == "backing":
		command = d.BuildBackingCommand()
	default:
		return errors.New("unknown action")
	}
	return d.Driver.QemuImg(command...)
}

func (d *ActionDisk) BuildCreateCommand() []string {
	command := []string{"create", "-f", d.Disk.Format}
	// add user-provided convert args
	command = append(command, d.QemuImgArgs.Create...)
	// add target path and size.
	command = append(command, d.Disk.FullPath, d.Disk.Size)
	return command
}

func (d *ActionDisk) BuildConvertCommand() []string {
	command := []string{"convert", "-O", d.Disk.Format}
	// add user-provided convert args
	command = append(command, d.QemuImgArgs.Convert...)
	// add target path and size.
	command = append(command, d.Disk.Source, d.Disk.FullPath)
	return command
}

func (d *ActionDisk) BuildBackingCommand() []string {
	command := []string{"create", "-f", d.Disk.Format}
	// add user-provided convert args
	command = append(command, d.QemuImgArgs.Create...)
	// add target path and backing option.
	command = append(command, "-b", d.Disk.Source, d.Disk.FullPath)
	return command
}

func (d *ActionDisk) RunCopyCommand() error {
	return d.Driver.Copy(d.Disk.Source, d.Disk.FullPath)
}

// This step creates the virtual disk that will be used as the
// hard drive for the virtual machine.
type stepCreateDisk struct {
	AdditionalDiskSize []string
	DiskImage          bool
	DisksOrder         []string
	DiskSize           string
	Format             string
	OutputDir          string
	UseBackingFile     bool
	VMName             string
	QemuImgArgs        QemuImgArgs
	ManyDisks          bool
}

func (s *stepCreateDisk) convertAction(disk Disk, driver Driver, ui packersdk.Ui) ActionDisk {
	ext := filepath.Ext(disk.Source)
	if len(ext) >= 1 && ext[1:] == s.Format && len(s.QemuImgArgs.Convert) == 0 {
		ui.Message("File extension already matches desired output format. " +
			"Skipping qemu-img convert step")
		return s.CopyAction(disk, driver)
	}
	return ActionDisk{
		disk,
		s.QemuImgArgs,
		driver,
		"convert",
	}
}

func (s *stepCreateDisk) createAction(disk Disk, driver Driver) ActionDisk {
	return ActionDisk{
		disk,
		s.QemuImgArgs,
		driver,
		"create",
	}
}

func (s *stepCreateDisk) backingAction(disk Disk, driver Driver) ActionDisk {
	return ActionDisk{
		disk,
		s.QemuImgArgs,
		driver,
		"backing",
	}
}

func (s *stepCreateDisk) CopyAction(disk Disk, driver Driver) ActionDisk {
	return ActionDisk{
		disk,
		s.QemuImgArgs,
		driver,
		"copy",
	}
}

func (s *stepCreateDisk) get_target_path(name string, index int) string {
	return filepath.Join(s.OutputDir, fmt.Sprintf("%s-%d", name, index))
}

func (s *stepCreateDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	name := s.VMName
	isoPath := state.Get("iso_path").(string)
	disksOrder := []string{isoPath}
	source_dir := ""

	ui.Say("Creating required virtual machine disks")
	actionDisks := []ActionDisk{}
	var action ActionDisk

	if s.ManyDisks {
		disksOrder = s.DisksOrder
		source_dir = isoPath
	}
	for i, d := range disksOrder {
		disk := Disk{
			FullPath: s.get_target_path(name, i),
			Source:   filepath.Join(source_dir, d),
			Format:   s.Format,
			Size:     s.DiskSize,
		}
		if s.DiskImage && s.UseBackingFile {
			action = s.backingAction(disk, driver)
		} else if s.DiskImage && !s.UseBackingFile {
			action = s.convertAction(disk, driver, ui)
		} else {
			action = s.createAction(disk, driver)
		}
		actionDisks = append(actionDisks, action)
	}

	// Additional disks
	nbBaseDisks := len(actionDisks)
	diskFullPaths := []string{}
	if len(s.AdditionalDiskSize) > 0 {
		for i, diskSize := range s.AdditionalDiskSize {
			action := s.createAction(Disk{
				FullPath: s.get_target_path(name, nbBaseDisks+i),
				Size:     diskSize,
				Format:   s.Format,
			}, driver)
			actionDisks = append(actionDisks, action)
		}
	}

	// Create all required disks
	for _, a := range actionDisks {
		log.Printf("[INFO] Creating/Copy disk with Path: %s and Size: %s", a.Disk.FullPath, a.Disk.Size)
		diskFullPaths = append(diskFullPaths, a.Disk.FullPath)

		if err := a.RunCommand(); err != nil {
			err := fmt.Errorf("error creating hard drive: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Stash the disk paths so we can retrieve later
	state.Put("qemu_disk_paths", diskFullPaths)

	return multistep.ActionContinue
}

func (s *stepCreateDisk) Cleanup(state multistep.StateBag) {}
