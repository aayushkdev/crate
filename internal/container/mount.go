package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Mount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"read_only,omitempty"`
}

type OpenedMount struct {
	Mount Mount
	File  *os.File
	Info  os.FileInfo
}

func ParseMounts(values []string) ([]Mount, error) {
	mounts := make([]Mount, 0, len(values))
	for _, value := range values {
		mount, err := ParseMount(value)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}

	return ValidateMounts(mounts)
}

func ParseMount(value string) (Mount, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return Mount{}, fmt.Errorf("invalid volume %q: expected HOST_PATH:CONTAINER_PATH[:ro]", value)
	}

	mount := Mount{
		Source:      parts[0],
		Destination: parts[1],
	}
	if len(parts) == 3 {
		switch parts[2] {
		case "ro":
			mount.ReadOnly = true
		case "rw", "":
		default:
			return Mount{}, fmt.Errorf("invalid volume %q: mode must be ro or rw", value)
		}
	}

	mounts, err := ValidateMounts([]Mount{mount})
	if err != nil {
		return Mount{}, err
	}

	return mounts[0], nil
}

func ValidateMounts(mounts []Mount) ([]Mount, error) {
	seen := make(map[string]struct{}, len(mounts))
	for i := range mounts {
		mount := &mounts[i]
		if strings.ContainsRune(mount.Source, 0) || strings.ContainsRune(mount.Destination, 0) {
			return nil, fmt.Errorf("invalid volume %q:%q: paths must not contain NUL", mount.Source, mount.Destination)
		}
		if !filepath.IsAbs(mount.Source) {
			return nil, fmt.Errorf("invalid volume source %q: must be an absolute path", mount.Source)
		}
		if !filepath.IsAbs(mount.Destination) {
			return nil, fmt.Errorf("invalid volume destination %q: must be an absolute path", mount.Destination)
		}

		mount.Source = filepath.Clean(mount.Source)
		mount.Destination = filepath.Clean(mount.Destination)
		if mount.Destination == "/" {
			return nil, fmt.Errorf("invalid volume destination %q: cannot mount over container root", mount.Destination)
		}
		if _, ok := seen[mount.Destination]; ok {
			return nil, fmt.Errorf("duplicate volume destination %q", mount.Destination)
		}
		seen[mount.Destination] = struct{}{}
	}

	return mounts, nil
}

func OpenMounts(mounts []Mount) ([]OpenedMount, error) {
	opened := make([]OpenedMount, 0, len(mounts))
	for _, mount := range mounts {
		file, err := os.Open(mount.Source)
		if err != nil {
			CloseMounts(opened)
			return nil, fmt.Errorf("open volume source %s: %w", mount.Source, err)
		}

		info, err := file.Stat()
		if err != nil {
			file.Close()
			CloseMounts(opened)
			return nil, fmt.Errorf("stat volume source %s: %w", mount.Source, err)
		}
		mode := info.Mode()
		if !mode.IsDir() && !mode.IsRegular() {
			file.Close()
			CloseMounts(opened)
			return nil, fmt.Errorf("volume source %s must be a regular file or directory", mount.Source)
		}

		opened = append(opened, OpenedMount{
			Mount: mount,
			File:  file,
			Info:  info,
		})
	}

	return opened, nil
}

func CloseMounts(mounts []OpenedMount) {
	for _, mount := range mounts {
		_ = mount.File.Close()
	}
}
