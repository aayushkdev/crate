package fs

import (
	"fmt"
	"os"
)

func OpenHostDevFDs(rootless bool) (map[string]*os.File, error) {

	devs := []string{
		"null",
		"zero",
		"random",
		"urandom",
		"full",
	}

	fds := make(map[string]*os.File, len(devs))

	for _, d := range devs {
		f, err := os.OpenFile("/dev/"+d, os.O_RDWR, 0)
		if err != nil {
			for _, of := range fds {
				of.Close()
			}
			return nil, fmt.Errorf("open host /dev/%s: %w", d, err)
		}
		fds[d] = f
	}

	return fds, nil
}

func CloseHostFDs(fds map[string]*os.File) {
	for _, f := range fds {
		_ = f.Close()
	}
}
