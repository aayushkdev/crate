package runtime

import (
	"fmt"
	"os"
	"strings"

	"github.com/aayushkdev/crate/internal/container"
)

func warnPrivilegeDropFailure(containerID string, exitCode int) {
	if exitCode != 127 {
		return
	}

	cfg, err := container.ReadConfig(containerID)
	if err != nil {
		return
	}
	if !cfg.Rootless || !container.IsRootUserSpec(cfg.User) {
		return
	}

	data, err := os.ReadFile(container.LogPath(containerID))
	if err != nil {
		return
	}
	if !strings.Contains(string(data), "setgroups failed") {
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"crate: warning: image %q failed to drop privileges in rootless mode (setgroups disabled). Try --user if the image can start as its final service user\n",
		cfg.Image,
	)
}
