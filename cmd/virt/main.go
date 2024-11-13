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
	defKernelPath    = "vmlinux"
	defInitramfsPath = "initramfs"
	defRootfs        = "rootfs"

	defCmdlineArgInitramfs = "root=/dev/ram0"
	defCmdlineArgRootfs    = "rootfstype=virtiofs root=rootfs"
)

var defCmdline = []string{
	"console=hvc0",
	"earlyprintk=serial,hvc0",
	"printk.devkmsg=on",
	"loglevel=7",
	"raid=noautodetect",
	"init=/init",
}

func main() {
	var cfg virt.Config
	cmd := &cobra.Command{
		Use:     "virt [flags]",
		Version: "v0.1",
		Long: `Launch a VM using macOS's Virtualization.framework.

virt can be invoked with no arguments, in which case it will try to detect
which vmlinux, initramfs / rootfs and cmdline should be used.

Both --initramfs and --rootfs are mutually exclusive. If none is provided, virt
will pick 'initramfs' if that file exists in the current directory. Otherwise,
it'll fall back to 'rootfs' (if that directory exists).

When --cmdline isn't provided, it'll automatically add the appropriate
arguments needed to boot using either the 'initramfs' or 'rootfs' provided.
That is, when --initramfs is provided (or auto-detected), it'll append
'root=/dev/ram0', and when --rootfs is provided (or auto-detected), it'll
append: 'rootfstype=virtiofs rootflags=trans=virtio,cache=mmap,msize=1048576'.

Note that, you can use --debug to show debug logs, including the default
--cmdline, with the various arguments added automatically.

Check ` + "`" + `man 7 bootparam` + "`" + ` if you're not sure which cmdline arguments to use.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := run(cfg); err != nil {
				fmt.Fprintln(os.Stderr, "ERROR:", err.Error())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&cfg.Kernel, "kernel", defKernelPath, "Path to the uncompressed kernel binary (eg. vmlinux)")
	cmd.Flags().StringVar(&cfg.Initramfs, "initramfs", "", "Path to the initramfs cpio archive ('initramfs' by default, mutually exclusive with --rootfs)")
	cmd.Flags().StringVar(&cfg.Rootfs, "rootfs", "", "Path to the rootfs directory ('rootfs' by default, mutually exclusive with --initramfs)")
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
	cmdline := []string{cfg.Cmdline}

	if exists, err := fileExists(cfg.Kernel); err != nil {
		return err
	} else if !exists {
		return errors.New("kernel not found")
	}

	if cfg.Initramfs != "" && cfg.Rootfs != "" {
		return errors.New("--initramfs and --rootfs are mutually exclusive")
	}

	var detectRoot bool
	if cfg.Initramfs == "" && cfg.Rootfs == "" {
		detectRoot = true
	}

	if cfg.Initramfs != "" || detectRoot {
		initramfs := cfg.Initramfs
		if initramfs == "" {
			initramfs = defInitramfsPath
		}

		exists, err := fileExists(initramfs)
		if err != nil {
			return err
		}

		if !exists && !detectRoot {
			return errors.New("initramfs not found")
		}
		if exists {
			cfg.Initramfs = initramfs
			cmdline = append(cmdline, defCmdlineArgInitramfs)
		}
	}

	if cfg.Rootfs != "" || detectRoot {
		rootfs := cfg.Rootfs
		if rootfs == "" {
			rootfs = defInitramfsPath
		}

		exists, err := dirExists(rootfs)
		if err != nil {
			return err
		}

		if !exists && !detectRoot {
			return errors.New("rootfs not found")
		}
		if exists {
			cfg.Rootfs = rootfs
			cmdline = append(cmdline, defCmdlineArgRootfs)
		}
	}

	if cfg.Initramfs == "" && cfg.Rootfs == "" {
		return errors.New("missing --initramfs or --rootfs -- no initramfs or rootfs detected")
	}

	cfg.Cmdline = strings.Join(cmdline, " ")
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

func dirExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.Mode().IsDir(), nil
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
