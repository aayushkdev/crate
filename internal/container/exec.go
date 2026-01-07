package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolvePath(cmd string, env []string) (string, error) {
	if strings.Contains(cmd, "/") {
		if st, err := os.Stat(cmd); err == nil && !st.IsDir() {
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
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			return full, nil
		}
	}

	return "", fmt.Errorf("executable %q not found in PATH", cmd)
}

func ResolveEntrypoint(cfg *Config, userCmd []string) ([]string, error) {
	switch {
	case len(cfg.EntryPoint) > 0 && len(userCmd) > 0:
		return append(cfg.EntryPoint, userCmd...), nil

	case len(cfg.EntryPoint) > 0 && len(cfg.Cmd) > 0:
		return append(cfg.EntryPoint, cfg.Cmd...), nil

	case len(cfg.EntryPoint) > 0:
		return cfg.EntryPoint, nil

	case len(userCmd) > 0:
		return userCmd, nil

	case len(cfg.Cmd) > 0:
		return cfg.Cmd, nil

	default:
		return nil, fmt.Errorf("no command specified")
	}
}
