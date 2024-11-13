package virt

import (
	"context"
	"fmt"
	"os"

	"github.com/Code-Hex/vz/v3"
)

type Config struct {
	Kernel    string
	Initramfs string
	Cmdline   string
}

func CreateVM(cfg Config) (*vz.VirtualMachine, error) {
	vmConfig, err := createVMConfig(cfg)
	if err != nil {
		return nil, err
	}

	vm, err := vz.NewVirtualMachine(vmConfig)
	if err != nil {
		return nil, fmt.Errorf("creating vm from its config: %w", err)
	}

	return vm, nil
}

func createVMConfig(cfg Config) (*vz.VirtualMachineConfiguration, error) {
	bootloader, err := vz.NewLinuxBootLoader(cfg.Kernel,
		vz.WithCommandLine(cfg.Cmdline),
		vz.WithInitrd(cfg.Initramfs),
	)
	if err != nil {
		return nil, fmt.Errorf("creating linux bootloader: %w", err)
	}

	vmConfig, err := vz.NewVirtualMachineConfiguration(bootloader, 4, 4*1024*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("creating vm config: %w", err)
	}

	consoleDeviceConfig, err := createConsoleConfig()
	if err != nil {
		return nil, err
	}

	vmConfig.SetSerialPortsVirtualMachineConfiguration([]*vz.VirtioConsoleDeviceSerialPortConfiguration{
		consoleDeviceConfig,
	})

	entropyDeviceConfig, err := vz.NewVirtioEntropyDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("creating entropy device config: %w", err)
	}

	vmConfig.SetEntropyDevicesVirtualMachineConfiguration([]*vz.VirtioEntropyDeviceConfiguration{
		entropyDeviceConfig,
	})

	if ok, err := vmConfig.Validate(); !ok || err != nil {
		return nil, fmt.Errorf("invalid vm config: %w", err)
	}

	return vmConfig, err
}

func createConsoleConfig() (*vz.VirtioConsoleDeviceSerialPortConfiguration, error) {
	consoleAttachment, err := vz.NewFileHandleSerialPortAttachment(os.Stdin, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("creating handle-based console attachment: %w", err)
	}

	consoleDeviceConfig, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(consoleAttachment)
	if err != nil {
		return nil, fmt.Errorf("creating console device config: %w", err)
	}

	return consoleDeviceConfig, err
}

func RunVM(ctx context.Context, vm *vz.VirtualMachine) error {
	if err := vm.Start(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			if vm.CanStop() {
				if err := vm.Stop(); err != nil {
					return err
				}
			}

			return nil
		case state := <-vm.StateChangedNotify():
			fmt.Printf("state change: %s\n", state)
		}
	}
}
