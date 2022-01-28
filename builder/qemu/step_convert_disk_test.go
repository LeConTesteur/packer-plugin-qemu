package qemu

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
)

func Test_buildConvertCommand(t *testing.T) {
	type testCase struct {
		Step     *stepConvertDisk
		Expected []string
		Reason   string
	}
	testcases := []testCase{
		{
			&stepConvertDisk{
				Format:          "qcow2",
				DiskCompression: false,
			},
			[]string{"convert", "-O", "qcow2", "source.qcow", "target.qcow2"},
			"Basic, happy path, no compression, no extra args",
		},
		{
			&stepConvertDisk{
				Format:          "qcow2",
				DiskCompression: true,
			},
			[]string{"convert", "-c", "-O", "qcow2", "source.qcow", "target.qcow2"},
			"Basic, happy path, with compression, no extra args",
		},
		{
			&stepConvertDisk{
				Format:          "qcow2",
				DiskCompression: true,
				QemuImgArgs: QemuImgArgs{
					Convert: []string{"-o", "preallocation=full"},
				},
			},
			[]string{"convert", "-c", "-o", "preallocation=full", "-O", "qcow2", "source.qcow", "target.qcow2"},
			"Basic, happy path, with compression, one set of extra args",
		},
	}

	for _, tc := range testcases {
		command := tc.Step.buildConvertCommand("source.qcow", "target.qcow2")

		assert.Equal(t, command, tc.Expected,
			fmt.Sprintf("%s. Expected %#v", tc.Reason, tc.Expected))
	}
}

func TeststepConvertDisk_Skips(t *testing.T) {
	testConfigs := []*Config{
		&Config{
			DiskCompression: false,
			SkipCompaction:  true,
		},
	}
	for _, config := range testConfigs {
		state := testState(t)
		driver := state.Get("driver").(*DriverMock)

		state.Put("config", config)
		state.Put("qemu_disk_paths", []string{})
		step := new(stepResizeDisk)

		// Test the run
		if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
			t.Fatalf("bad action: %#v", action)
		}
		if _, ok := state.GetOk("error"); ok {
			t.Fatal("should NOT have error")
		}
		if len(driver.QemuImgCalls) > 0 {
			t.Fatal("should NOT have called qemu-img")
		}
	}
}

func TeststepConvertDisk_Run(t *testing.T) {
	type testCase struct {
		Step         *stepConvertDisk
		DisksPath    []string
		QemuExpected []string
		Reason       string
	}
	testcases := []testCase{
		{
			&stepConvertDisk{},
			nil,
			nil,
			"",
		},
		{
			&stepConvertDisk{
				Format: "qcow2",
			},
			[]string{"output/target-0"},
			[]string{"resize", "-f", "qcow2", "output/target-0", "1234M"},
			"",
		},
		{
			&stepConvertDisk{
				Format: "qcow2",
				QemuImgArgs: QemuImgArgs{
					Resize: []string{"-foo", "-bar"},
				},
			},
			[]string{"output/target-0", "output/target-1", "output/target-2"},
			[]string{
				"resize", "-f", "qcow2", "-foo", "-bar", "output/target-0", "1234M",
				"resize", "-f", "qcow2", "-foo", "-bar", "output/target-1", "1234M",
				"resize", "-f", "qcow2", "-foo", "-bar", "output/target-2", "1234M",
			},
			"",
		},
	}
	for _, tc := range testcases {
		d := new(DriverMock)
		state := copyTestState(t, d)
		state.Put("qemu_disk_paths", tc.DisksPath)

		// Test the run
		action := tc.Step.Run(context.TODO(), state)

		if action != multistep.ActionContinue {
			t.Fatalf("Should have gotten an ActionContinue")
		}
		assert.Equal(t, tc.QemuExpected, d.QemuImgCalls,
			fmt.Sprintf("%s. Expected %#v", tc.Reason, tc.QemuExpected))
	}
}
