package qemu

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
)

func Test_buildCreateCommand(t *testing.T) {
	type testCase struct {
		Action       *ActionDisk
		CopyExpected []string
		QemuExpected []string
		Reason       string
	}
	testcases := []testCase{
		{
			&ActionDisk{
				Disk: Disk{
					FullPath: "target.qcow2",
					Size:     "1234M",
					Source:   "source.qcow2",
					Format:   "qcow2",
				},
				ActionType: "create",
			},
			nil,
			[]string{"create", "-f", "qcow2", "target.qcow2", "1234M"},
			"Basic, happy path, no backing store, no extra args",
		},
		{
			&ActionDisk{
				Disk: Disk{
					FullPath: "target.qcow2",
					Size:     "1234M",
					Source:   "source.qcow2",
					Format:   "qcow2",
				},
				ActionType: "backing",
			},
			nil,
			[]string{"create", "-f", "qcow2", "-b", "source.qcow2", "target.qcow2"},
			"Basic, happy path, backing store",
		},
		{
			&ActionDisk{
				Disk: Disk{
					FullPath: "target.qcow2",
					Size:     "1234M",
					Source:   "source.qcow2",
					Format:   "qcow2",
				},
				ActionType: "copy",
			},
			[]string{"source.qcow2", "target.qcow2"},
			nil,
			"Basic, happy path, simple copy",
		},
		{
			&ActionDisk{
				Disk: Disk{
					FullPath: "target.qcow2",
					Size:     "1234M",
					Source:   "source.vmdk",
					Format:   "qcow2",
				},
				ActionType: "convert",
			},
			nil,
			[]string{"convert", "-O", "qcow2", "source.vmdk", "target.qcow2"},
			"Basic, happy path, convert",
		},
		{
			&ActionDisk{
				Disk: Disk{
					FullPath: "target.qcow2",
					Size:     "1234M",
					Source:   "source.qcow2",
					Format:   "qcow2",
				},
				QemuImgArgs: QemuImgArgs{
					Create: []string{"-foo", "-bar"},
				},
				ActionType: "create",
			},
			nil,
			[]string{"create", "-f", "qcow2", "-foo", "-bar", "target.qcow2", "1234M"},
			"Basic, happy path, no backing store, extra args",
		},
		{
			&ActionDisk{
				Disk: Disk{
					FullPath: "target.qcow2",
					Size:     "1234M",
					Source:   "source.qcow2",
					Format:   "qcow2",
				},
				QemuImgArgs: QemuImgArgs{
					Create: []string{"-foo", "-bar"},
				},
				ActionType: "backing",
			},
			nil,
			[]string{"create", "-f", "qcow2", "-foo", "-bar", "-b", "source.qcow2", "target.qcow2"},
			"Basic, happy path, backing store, extra args",
		},
		{
			&ActionDisk{
				Disk: Disk{
					FullPath: "target.qcow2",
					Size:     "1234M",
					Source:   "source.qcow2",
					Format:   "qcow2",
				},
				QemuImgArgs: QemuImgArgs{
					Convert: []string{"-foo", "-bar"},
				},
				ActionType: "convert",
			},
			nil,
			[]string{"convert", "-O", "qcow2", "-foo", "-bar", "source.qcow2", "target.qcow2"},
			"Basic, happy path, convert, extra args",
		},
	}

	for _, tc := range testcases {
		d := new(DriverMock)
		tc.Action.Driver = d
		err := tc.Action.RunCommand()
		assert.Nil(t, err)
		assert.Equal(t, tc.CopyExpected, d.CopyCalls,
			fmt.Sprintf("%s. Expected %#v", tc.Reason, tc.CopyExpected))
		assert.Equal(t, tc.QemuExpected, d.QemuImgCalls,
			fmt.Sprintf("%s. Expected %#v", tc.Reason, tc.QemuExpected))
	}
}

func Test_StepCreateCalled(t *testing.T) {
	type testCase struct {
		Step         *stepCreateDisk
		QemuDisks    []string
		CopyExpected []string
		QemuExpected []string
		Reason       string
	}
	testcases := []testCase{
		{
			&stepCreateDisk{
				Format:         "qcow2",
				DiskImage:      true,
				DiskSize:       "1M",
				VMName:         "target",
				OutputDir:      "output",
				UseBackingFile: true,
			},
			nil,
			nil,
			[]string{
				"create", "-f", "qcow2", "-b", "source.qcow2", "output/target-0",
			},
			"Basic, happy path, backing store, no additional disks, outputdir set",
		},
		{
			&stepCreateDisk{
				Format:         "raw",
				DiskImage:      false,
				DiskSize:       "4M",
				VMName:         "target",
				UseBackingFile: false,
			},
			nil,
			nil,
			[]string{
				"create", "-f", "raw", "target-0", "4M",
			},
			"Basic, happy path, raw, no additional disks",
		},
		{
			&stepCreateDisk{
				Format:             "qcow2",
				DiskImage:          true,
				DiskSize:           "4M",
				VMName:             "target",
				UseBackingFile:     false,
				AdditionalDiskSize: []string{"3M", "8M"},
			},
			nil,
			[]string{"source.qcow2", "target-0"},
			[]string{
				"create", "-f", "qcow2", "target-1", "3M",
				"create", "-f", "qcow2", "target-2", "8M",
			},
			"Skips disk creation when disk can be copied, with additional disks",
		},
		{
			&stepCreateDisk{
				Format:         "qcow2",
				DiskImage:      true,
				DiskSize:       "4M",
				VMName:         "target",
				OutputDir:      "output",
				UseBackingFile: false,
			},
			nil,
			[]string{"source.qcow2", "output/target-0"},
			nil,
			"Skips disk creation when disk can be copied, outputdir set",
		},
		{
			&stepCreateDisk{
				Format:             "qcow2",
				DiskImage:          true,
				DiskSize:           "1M",
				VMName:             "target",
				UseBackingFile:     true,
				AdditionalDiskSize: []string{"3M", "8M"},
			},
			nil,
			nil,
			[]string{
				"create", "-f", "qcow2", "-b", "source.qcow2", "target-0",
				"create", "-f", "qcow2", "target-1", "3M",
				"create", "-f", "qcow2", "target-2", "8M",
			},
			"Basic, happy path, backing store, additional disks",
		},
		{
			&stepCreateDisk{
				Format:         "qcow2",
				DiskImage:      true,
				VMName:         "target",
				UseBackingFile: true,
				ManyDisks:      true,
			},
			[]string{
				"source.qcow2.extract",
			},
			nil,
			[]string{
				"create", "-f", "qcow2", "-b", "output_archive/source.qcow2.extract", "target-0",
			},
			"Basic, happy path, backing store, no additional disks, with many disks in input",
		},

		{
			&stepCreateDisk{
				Format:         "qcow2",
				DiskImage:      true,
				VMName:         "target",
				UseBackingFile: true,
				ManyDisks:      true,
			},
			[]string{
				"source0.qcow2.extract", "source1.qcow2.extract", "source2.qcow2.extract",
			},
			nil,
			[]string{
				"create", "-f", "qcow2", "-b", "output_archive/source0.qcow2.extract", "target-0",
				"create", "-f", "qcow2", "-b", "output_archive/source1.qcow2.extract", "target-1",
				"create", "-f", "qcow2", "-b", "output_archive/source2.qcow2.extract", "target-2",
			},
			"Basic, happy path, backing store, no additional disks, with many disks in input",
		},
		{
			&stepCreateDisk{
				Format:         "raw",
				DiskImage:      false,
				DiskSize:       "4M",
				VMName:         "target",
				UseBackingFile: false,
				ManyDisks:      true,
			},
			[]string{
				"source.qcow2.extract",
			},
			nil,
			[]string{
				"create", "-f", "raw", "target-0", "4M",
			},
			"Basic, happy path, raw, no additional disks, with many disks in input",
		},
		{
			&stepCreateDisk{
				Format:             "qcow2",
				DiskImage:          true,
				DiskSize:           "4M",
				VMName:             "target",
				UseBackingFile:     false,
				AdditionalDiskSize: []string{"3M", "8M"},
				ManyDisks:          true,
			},
			[]string{
				"source0.qcow2.extract", "source1.qcow2.extract", "source2.qcow2.extract",
			},
			nil,
			[]string{
				"convert", "-O", "qcow2", "output_archive/source0.qcow2.extract", "target-0",
				"convert", "-O", "qcow2", "output_archive/source1.qcow2.extract", "target-1",
				"convert", "-O", "qcow2", "output_archive/source2.qcow2.extract", "target-2",
				"create", "-f", "qcow2", "target-3", "3M",
				"create", "-f", "qcow2", "target-4", "8M",
			},
			"Skips disk creation when disk can be copied, with many disks in input",
		},
		{
			&stepCreateDisk{
				Format:             "qcow2",
				DiskImage:          true,
				DiskSize:           "4M",
				VMName:             "target",
				UseBackingFile:     false,
				AdditionalDiskSize: []string{"3M", "8M"},
				OutputDir:          "output",
				ManyDisks:          true,
			},
			[]string{
				"source0.qcow2.extract",
			},
			nil,
			[]string{
				"convert", "-O", "qcow2", "output_archive/source0.qcow2.extract", "output/target-0",
				"create", "-f", "qcow2", "output/target-1", "3M",
				"create", "-f", "qcow2", "output/target-2", "8M",
			},
			"Skips disk creation when disk can be copied, with many disks in input, outputdir set",
		},
		{
			&stepCreateDisk{
				Format:         "qcow2",
				DiskImage:      true,
				DiskSize:       "4M",
				VMName:         "target",
				UseBackingFile: false,
				OutputDir:      "output",
				ManyDisks:      true,
			},
			[]string{
				"source0.qcow2", "source1.qcow2", "source2.qcow2",
			},
			[]string{
				"output_archive/source0.qcow2", "output/target-0",
				"output_archive/source1.qcow2", "output/target-1",
				"output_archive/source2.qcow2", "output/target-2",
			},
			nil,
			"Skips disk creation when disk can be copied, with many disks in input, outputdir set",
		},
		{
			&stepCreateDisk{
				Format:             "qcow2",
				DiskImage:          true,
				DiskSize:           "1M",
				VMName:             "target",
				UseBackingFile:     true,
				AdditionalDiskSize: []string{"3M", "8M"},
				ManyDisks:          true,
			},
			[]string{
				"source0.qcow2.extract", "source1.qcow2.extract", "source2.qcow2.extract",
			},
			nil,
			[]string{
				"create", "-f", "qcow2", "-b", "output_archive/source0.qcow2.extract", "target-0",
				"create", "-f", "qcow2", "-b", "output_archive/source1.qcow2.extract", "target-1",
				"create", "-f", "qcow2", "-b", "output_archive/source2.qcow2.extract", "target-2",
				"create", "-f", "qcow2", "target-3", "3M",
				"create", "-f", "qcow2", "target-4", "8M",
			},
			"Basic, happy path, backing store, additional disks, with many disks in input",
		},
	}

	for _, tc := range testcases {
		d := new(DriverMock)
		state := copyTestState(t, d)
		if tc.Step.ManyDisks {
			state.Put("iso_path", "output_archive")
		} else {
			state.Put("iso_path", "source.qcow2")
		}
		tc.Step.DisksOrder = tc.QemuDisks
		action := tc.Step.Run(context.TODO(), state)

		if action != multistep.ActionContinue {
			t.Fatalf("Should have gotten an ActionContinue")
		}
		assert.Equal(t, tc.CopyExpected, d.CopyCalls,
			fmt.Sprintf("%s. Expected %#v", tc.Reason, tc.CopyExpected))
		assert.Equal(t, tc.QemuExpected, d.QemuImgCalls,
			fmt.Sprintf("%s. Expected %#v", tc.Reason, tc.QemuExpected))
	}
}
