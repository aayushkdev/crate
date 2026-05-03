package net

import (
	"os"
	"path/filepath"
	"strings"
)

func prepareRootfsFiles(rootfs string) error {
	if err := writeResolvConf(rootfs); err != nil {
		return err
	}

	return writeHosts(rootfs)
}

func writeResolvConf(rootfs string) error {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return err
	}

	path := filepath.Join(rootfs, "etc", "resolv.conf")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func writeHosts(rootfs string) error {
	path := filepath.Join(rootfs, "etc", "hosts")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	content := strings.Join([]string{
		"127.0.0.1 localhost",
		"::1 localhost ip6-localhost ip6-loopback",
		"127.0.0.1 crate",
		"",
	}, "\n")

	return os.WriteFile(path, []byte(content), 0644)
}
