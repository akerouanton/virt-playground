package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/Code-Hex/vz/v3"
	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

const (
	kernelPath = "build/vmlinux"
	initrdPath = "build/initramfs"
)

var cmdLine = []string{
	"console=hvc0",
	"root=/dev/ram0",
}

// https://developer.apple.com/documentation/virtualization/running_linux_in_a_virtual_machine?language=objc#:~:text=Configure%20the%20Serial%20Port%20Device%20for%20Standard%20In%20and%20Out
func setRawMode(f *os.File) {
	var attr unix.Termios

	// Get settings for terminal
	termios.Tcgetattr(f.Fd(), &attr)

	// Put stdin into raw mode, disabling local echo, input canonicalization,
	// and CR-NL mapping.
	attr.Iflag &^= syscall.ICRNL
	attr.Lflag &^= syscall.ICANON | syscall.ECHO

	// Set minimum characters when reading = 1 char
	attr.Cc[syscall.VMIN] = 1

	// set timeout when reading as non-canonical mode
	attr.Cc[syscall.VTIME] = 0

	// reflects the changed settings
	termios.Tcsetattr(f.Fd(), termios.TCSANOW, &attr)
}

func main() {
	vm, err := createVM()
	if err != nil {
		panic(err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGTERM)

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(2)
	go func() {
		<-sigch

		cancel()
		wg.Done()
	}()

	go func() {
		runVM(ctx, vm)
		wg.Done()
	}()

	wg.Wait()
}

func runVM(ctx context.Context, vm *vz.VirtualMachine) {
	setRawMode(os.Stdin)

	if err := vm.Start(); err != nil {
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			if vm.CanStop() {
				if err := vm.Stop(); err != nil {
					panic(err)
				}
			}

			return
		case state := <-vm.StateChangedNotify():
			fmt.Printf("state change: %s\n", state)
		}
	}
}

func createVM() (*vz.VirtualMachine, error) {
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

	consoleDeviceConfig, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(consoleAttachment)
	if err != nil {
		return nil, fmt.Errorf("creating console device config: %w", err)
	}

	vmConfig.SetSerialPortsVirtualMachineConfiguration([]*vz.VirtioConsoleDeviceSerialPortConfiguration{
		consoleDeviceConfig,
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
