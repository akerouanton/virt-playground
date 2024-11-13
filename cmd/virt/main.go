package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/akerouanton/virt-playground/pkg/virt"
	"github.com/pkg/term/termios"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

const (
	defKernelPath    = "build/vmlinux"
	defInitramfsPath = "build/initramfs"
)

var defCmdline = []string{
	"console=hvc0",
	"root=/dev/ram0",
	"earlyprintk=serial,hvc0",
	"printk.devkmsg=on",
	"loglevel=7",
	"raid=noautodetect",
	"init=/init",
}

func main() {
	var cfg virt.Config
	cmd := &cobra.Command{
		Use:     "Launch a VM using macOS's Virtualization.framework",
		Version: "v0.1",
		Run: func(cmd *cobra.Command, args []string) {
			if err := run(cfg); err != nil {
				fmt.Fprintln(os.Stderr, "ERROR:", err.Error())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&cfg.Kernel, "kernel", defKernelPath, "Path to the uncompressed kernel file")
	cmd.Flags().StringVar(&cfg.Initramfs, "initramfs", defInitramfsPath, "Path to the kernel file (eg. initramfs)")
	cmd.Flags().StringVar(&cfg.Cmdline, "cmdline", strings.Join(defCmdline, " "), "Kernel cmdline")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfg virt.Config) error {
	if err := validateConfig(&cfg); err != nil {
		return err
	}

	vm, err := virt.CreateVM(cfg)
	if err != nil {
		return err
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-sigch
		cancel()
		return nil
	})

	eg.Go(func() error {
		setRawMode(os.Stdin)
		return virt.RunVM(ctx, vm)
	})

	return eg.Wait()
}

func validateConfig(cfg *virt.Config) error {
	var cmdline []string
	copy(cmdline, defCmdline)

	if exists, err := fileExists(cfg.Kernel); err != nil {
		return err
	} else if !exists {
		return errors.New("kernel not found")
	}

	if cfg.Initramfs != "" {
		exists, err := fileExists(cfg.Initramfs)
		if err != nil {
			return err
		}

		if !exists {
			return errors.New("initrd not found")
		}
	}

	if cfg.Initramfs == "" {
		return errors.New("missing --initramfs")
	}

	if cfg.Cmdline == "" {
		cfg.Cmdline = strings.Join(cmdline, " ")
	}

	return nil
}

func fileExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.Mode().IsRegular(), nil
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
