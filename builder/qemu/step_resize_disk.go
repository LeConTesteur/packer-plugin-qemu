package qemu

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// This step resizes the virtual disk that will be used as the
// hard drive for the virtual machine.
type stepResizeDisk struct {
	DiskImage      bool
	Format         string
	SkipResizeDisk bool
	DiskSize       string
	QemuImgArgs    QemuImgArgs
}

func (s *stepResizeDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	diskFullPaths := state.Get("qemu_disk_paths").([]string)

	if s.DiskImage == false || s.SkipResizeDisk == true {
		return multistep.ActionContinue
	}

	for _, path := range diskFullPaths {
		command := s.buildResizeCommand(path)

		ui.Say(fmt.Sprintf("Resizing hard drive %s ...", path))
		if err := driver.QemuImg(command...); err != nil {
			err := fmt.Errorf("Error creating hard drive: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

func (s *stepResizeDisk) buildResizeCommand(path string) []string {
	command := []string{"resize", "-f", s.Format}

	// add user-provided convert args
	command = append(command, s.QemuImgArgs.Resize...)

	// Add file and size
	command = append(command, path, s.DiskSize)

	return command
}

func (s *stepResizeDisk) Cleanup(state multistep.StateBag) {}
