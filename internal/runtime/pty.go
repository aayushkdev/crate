package runtime

import (
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func startAttached(cmd *exec.Cmd) (*os.File, error) {
	attrs := cmd.SysProcAttr
	attrs.Setsid = true
	attrs.Setctty = true

	ptmx, err := pty.StartWithAttrs(cmd, nil, attrs)
	if err != nil {
		return nil, err
	}

	return ptmx, nil
}

func relayAttached(ptmx *os.File, out io.Writer) error {
	defer ptmx.Close()

	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer term.Restore(int(os.Stdin.Fd()), oldState)
		}

		resize := make(chan os.Signal, 1)
		signal.Notify(resize, syscall.SIGWINCH)
		defer func() {
			signal.Stop(resize)
			close(resize)
		}()

		go func() {
			for range resize {
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					log.Printf("error resizing pty: %v", err)
				}
			}
		}()

		if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
			log.Printf("error resizing pty: %v", err)
		}
	}

	go io.Copy(ptmx, os.Stdin)
	_, err := io.Copy(out, ptmx)
	if err != nil && err != io.EOF && !errors.Is(err, syscall.EIO) {
		return err
	}

	return nil
}
