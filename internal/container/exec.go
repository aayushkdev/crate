package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolvePath(cmd string, env []string) (string, error) {
	if strings.Contains(cmd, "/") {
		if st, err := os.Stat(cmd); err == nil && !st.IsDir() && st.Mode()&0111 != 0 {
			return cmd, nil
		}
		return "", fmt.Errorf("executable %q not found", cmd)
	}

	path := "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			path = strings.TrimPrefix(e, "PATH=")
			break
		}
	}

	for _, dir := range strings.Split(path, ":") {
		full := filepath.Join(dir, cmd)
		if st, err := os.Stat(full); err == nil && !st.IsDir() && st.Mode()&0111 != 0 {
			return full, nil
		}
	}

	return "", fmt.Errorf("executable %q not found in PATH", cmd)
}
