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
	Rootfs    string
	RootfsRW  bool
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
	bootloaderOpts := []vz.LinuxBootLoaderOption{vz.WithCommandLine(cfg.Cmdline)}
	if cfg.Initramfs != "" {
		bootloaderOpts = append(bootloaderOpts, vz.WithInitrd(cfg.Initramfs))
	}

	bootloader, err := vz.NewLinuxBootLoader(cfg.Kernel, bootloaderOpts...)
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

	if cfg.Rootfs != "" {
		sharedDir, err := vz.NewSharedDirectory(cfg.Rootfs, !cfg.RootfsRW)
		if err != nil {
			return nil, fmt.Errorf("instantiating shared directory: %w", err)
		}

		singleDirShare, err := vz.NewSingleDirectoryShare(sharedDir)
		if err != nil {
			return nil, fmt.Errorf("instantiating single directory share: %w", err)
		}

		virtiofs, err := vz.NewVirtioFileSystemDeviceConfiguration("rootfs")
		if err != nil {
			return nil, err
		}
		virtiofs.SetDirectoryShare(singleDirShare)
		vmConfig.SetDirectorySharingDevicesVirtualMachineConfiguration([]vz.DirectorySharingDeviceConfiguration{virtiofs})
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
