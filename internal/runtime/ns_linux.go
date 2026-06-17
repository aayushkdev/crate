package runtime

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type namespaceFiles struct {
	user *os.File
	mnt  *os.File
	uts  *os.File
	net  *os.File
	pid  *os.File
}

func openContainerNamespaces(pid int, rootless bool, joinNet bool) (*namespaceFiles, error) {
	ns := &namespaceFiles{}
	var err error
	defer func() {
		if err != nil {
			ns.Close()
		}
	}()

	if rootless {
		ns.user, err = openNamespace(pid, "user")
		if err != nil {
			return nil, err
		}
	}
	ns.mnt, err = openNamespace(pid, "mnt")
	if err != nil {
		return nil, err
	}
	ns.uts, err = openNamespace(pid, "uts")
	if err != nil {
		return nil, err
	}
	if joinNet {
		ns.net, err = openNamespace(pid, "net")
		if err != nil {
			return nil, err
		}
	}
	ns.pid, err = openNamespace(pid, "pid")
	if err != nil {
		return nil, err
	}

	return ns, nil
}

func openNamespace(pid int, name string) (*os.File, error) {
	path := filepath.Join("/proc", fmt.Sprint(pid), "ns", name)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open namespace %s: %w", path, err)
	}

	return f, nil
}

func joinContainerNamespaces(ns *namespaceFiles) error {
	if ns.user != nil {
		if err := setnsFile(ns.user, unix.CLONE_NEWUSER); err != nil {
			return err
		}
	}
	if err := setnsFile(ns.mnt, unix.CLONE_NEWNS); err != nil {
		return err
	}
	if err := setnsFile(ns.uts, unix.CLONE_NEWUTS); err != nil {
		return err
	}
	if ns.net != nil {
		if err := setnsFile(ns.net, unix.CLONE_NEWNET); err != nil {
			return err
		}
	}
	if err := setnsFile(ns.pid, unix.CLONE_NEWPID); err != nil {
		return err
	}

	return nil
}

func setnsFile(f *os.File, nstype int) error {
	if err := unix.Setns(int(f.Fd()), nstype); err != nil {
		return fmt.Errorf("setns %s: %w", f.Name(), err)
	}

	return nil
}

func (ns *namespaceFiles) Close() {
	if ns == nil {
		return
	}
	for _, f := range []*os.File{ns.user, ns.mnt, ns.uts, ns.net, ns.pid} {
		if f != nil {
			_ = f.Close()
		}
	}
}
