package qemu

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
)

func TestStepResizeDisk_Skips(t *testing.T) {
	testConfigs := []*Config{
		&Config{
			DiskImage:      false,
			SkipResizeDisk: false,
		},
		&Config{
			DiskImage:      false,
			SkipResizeDisk: true,
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

func TestStepResizeDisk_Run(t *testing.T) {
	type testCase struct {
		Step         *stepResizeDisk
		DisksPath    []string
		QemuExpected []string
		Reason       string
	}
	testcases := []testCase{
		{
			&stepResizeDisk{},
			nil,
			nil,
			"",
		},
		{
			&stepResizeDisk{
				DiskImage:      true,
				SkipResizeDisk: false,
				DiskSize:       "1234M",
				Format:         "qcow2",
			},
			[]string{"output/target-0"},
			[]string{"resize", "-f", "qcow2", "output/target-0", "1234M"},
			"",
		},
		{
			&stepResizeDisk{
				DiskImage:      true,
				SkipResizeDisk: false,
				DiskSize:       "1234M",
				Format:         "qcow2",
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

func Test_buildResizeCommand(t *testing.T) {
	type testCase struct {
		Step     *stepResizeDisk
		Expected []string
		Reason   string
	}
	testcases := []testCase{
		{
			&stepResizeDisk{
				Format:   "qcow2",
				DiskSize: "1234M",
			},
			[]string{"resize", "-f", "qcow2", "source.qcow", "1234M"},
			"no extra args",
		},
		{
			&stepResizeDisk{
				Format:   "qcow2",
				DiskSize: "1234M",
				QemuImgArgs: QemuImgArgs{
					Resize: []string{"-foo", "bar"},
				},
			},
			[]string{"resize", "-f", "qcow2", "-foo", "bar", "source.qcow", "1234M"},
			"one set of extra args",
		},
	}

	for _, tc := range testcases {
		command := tc.Step.buildResizeCommand("source.qcow")

		assert.Equal(t, command, tc.Expected,
			fmt.Sprintf("%s. Expected %#v", tc.Reason, tc.Expected))
	}
}
