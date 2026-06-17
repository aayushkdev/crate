package runtime

import "os/exec"

func selfCommand(name string, containerID string, args []string) *exec.Cmd {
	argv := append([]string{name, containerID}, args...)
	return exec.Command("/proc/self/exe", argv...)
}
