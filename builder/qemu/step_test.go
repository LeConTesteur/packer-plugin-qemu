package qemu

import (
	"bytes"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func testState(t *testing.T) multistep.StateBag {
	state := new(multistep.BasicStateBag)
	state.Put("driver", new(DriverMock))
	state.Put("ui", &packersdk.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	})
	return state
}

func copyTestState(t *testing.T, d *DriverMock) multistep.StateBag {
	state := new(multistep.BasicStateBag)
	state.Put("ui", packersdk.TestUi(t))
	state.Put("driver", d)
	state.Put("iso_path", "example_source.qcow2")

	return state
}
