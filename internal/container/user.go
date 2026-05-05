package container

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type UserSpec struct {
	UID int
	GID int
}

func ResolveUser(spec string) (*UserSpec, error) {
	if spec == "" {
		return nil, nil
	}

	userPart, groupPart := splitUserSpec(spec)

	uid, defaultGID, err := resolveUserPart(userPart)
	if err != nil {
		return nil, err
	}

	gid := defaultGID
	if groupPart != "" {
		gid, err = resolveGroupPart(groupPart)
		if err != nil {
			return nil, err
		}
	}

	return &UserSpec{
		UID: uid,
		GID: gid,
	}, nil
}

func ApplyUser(spec string) error {
	user, err := ResolveUser(spec)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	if err := syscall.Setgid(user.GID); err != nil {
		return fmt.Errorf("setgid %d: %w", user.GID, err)
	}
	if err := syscall.Setuid(user.UID); err != nil {
		return fmt.Errorf("setuid %d: %w", user.UID, err)
	}

	return nil
}

func RootlessUserWarning(cfg *Config) string {
	if !cfg.Rootless {
		return ""
	}
	if cfg.User == "" || cfg.User == "0" || cfg.User == "root" || strings.HasPrefix(cfg.User, "0:") || strings.HasPrefix(cfg.User, "root:") {
		return fmt.Sprintf(
			"image %q starts as root in rootless mode; privilege-drop entrypoints may fail after startup because setgroups(2) is disabled. Try --user if the image can start as its final service user",
			cfg.Image,
		)
	}

	return ""
}

func splitUserSpec(spec string) (string, string) {
	userPart, groupPart, found := strings.Cut(spec, ":")
	if !found {
		return spec, ""
	}
	return userPart, groupPart
}

func resolveUserPart(value string) (int, int, error) {
	if value == "" {
		return 0, 0, fmt.Errorf("empty user in user spec")
	}
	if uid, err := strconv.Atoi(value); err == nil {
		gid, err := lookupPrimaryGIDByUID(uid)
		if err != nil {
			return uid, uid, nil
		}
		return uid, gid, nil
	}

	return lookupUserByName(value)
}

func resolveGroupPart(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("empty group in user spec")
	}
	if gid, err := strconv.Atoi(value); err == nil {
		return gid, nil
	}

	return lookupGroupByName(value)
}

func lookupUserByName(name string) (int, int, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return 0, 0, fmt.Errorf("open /etc/passwd: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 4 || parts[0] != name {
			continue
		}

		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, fmt.Errorf("parse uid for user %q: %w", name, err)
		}
		gid, err := strconv.Atoi(parts[3])
		if err != nil {
			return 0, 0, fmt.Errorf("parse gid for user %q: %w", name, err)
		}

		return uid, gid, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("read /etc/passwd: %w", err)
	}

	return 0, 0, fmt.Errorf("user %q not found in /etc/passwd", name)
}

func lookupPrimaryGIDByUID(uid int) (int, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return 0, fmt.Errorf("open /etc/passwd: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 4 {
			continue
		}

		entryUID, err := strconv.Atoi(parts[2])
		if err != nil || entryUID != uid {
			continue
		}

		gid, err := strconv.Atoi(parts[3])
		if err != nil {
			return 0, fmt.Errorf("parse gid for uid %d: %w", uid, err)
		}

		return gid, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("read /etc/passwd: %w", err)
	}

	return 0, fmt.Errorf("uid %d not found in /etc/passwd", uid)
}

func lookupGroupByName(name string) (int, error) {
	f, err := os.Open("/etc/group")
	if err != nil {
		return 0, fmt.Errorf("open /etc/group: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 3 || parts[0] != name {
			continue
		}

		gid, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("parse gid for group %q: %w", name, err)
		}

		return gid, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("read /etc/group: %w", err)
	}

	return 0, fmt.Errorf("group %q not found in /etc/group", name)
}
