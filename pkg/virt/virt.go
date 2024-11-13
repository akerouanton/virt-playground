package virt

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Code-Hex/vz/v3"
)

const (
	kernelPath = "build/vmlinux"
	initrdPath = "build/initramfs"
)

var cmdLine = []string{
	"console=hvc0",
	"root=/dev/ram0",
	"earlyprintk=serial,hvc0",
	"printk.devkmsg=on",
	"loglevel=7",
	"raid=noautodetect",
	"init=/init",
}

func CreateVM() (*vz.VirtualMachine, error) {
	vmConfig, err := createVMConfig()
	if err != nil {
		return nil, err
	}

	vm, err := vz.NewVirtualMachine(vmConfig)
	if err != nil {
		return nil, fmt.Errorf("creating vm from its config: %w", err)
	}

	return vm, nil
}

func createVMConfig() (*vz.VirtualMachineConfiguration, error) {
	bootloader, err := vz.NewLinuxBootLoader(kernelPath,
		vz.WithCommandLine(strings.Join(cmdLine, " ")),
		vz.WithInitrd(initrdPath),
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
