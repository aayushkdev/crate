package net

import (
	"bufio"
	"os"
	"strings"
)

func interfaceExists(name string) bool {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, name+":") {
			return true
		}
	}

	return false
}
