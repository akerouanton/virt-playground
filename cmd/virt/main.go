package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/akerouanton/virt-playground/pkg/virt"
	"github.com/pkg/term/termios"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

func main() {
	vm, err := virt.CreateVM()
	if err != nil {
		panic(err)
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

	if err := eg.Wait(); err != nil {
		panic(err)
	}
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
