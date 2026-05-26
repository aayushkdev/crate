package container

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	storage "github.com/aayushkdev/crate/internal/storage"
)

const maxHostnameBytes = 63

func resolveContainerName(requested string) (string, error) {
	name := strings.TrimSpace(requested)
	if name == "" {
		name = randomName()
	}
	name = sanitizeName(name)
	if name == "" {
		return "", fmt.Errorf("invalid container name")
	}

	for {
		exists, err := nameExists(name)
		if err != nil {
			return "", err
		}
		if !exists {
			return name, nil
		}
		if requested != "" {
			return "", fmt.Errorf("container name %q already in use", name)
		}
		name = randomName()
		name = sanitizeName(name)
		if name == "" {
			return "", fmt.Errorf("invalid container name")
		}
	}
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	b.Grow(len(name))
	for i := 0; i < len(name); i++ {
		ch := name[i]
		switch {
		case ch >= 'a' && ch <= 'z':
			b.WriteByte(ch)
		case ch >= '0' && ch <= '9':
			b.WriteByte(ch)
		case ch == '-' || ch == '_' || ch == ' ':
			b.WriteByte('-')
		default:
			b.WriteByte('-')
		}
	}
	name = strings.Trim(b.String(), "-")
	if name == "" {
		return ""
	}
	name = ClampHostname(name)
	return name
}

func randomName() string {
	adj := nameAdjectives[randomIndex(len(nameAdjectives))]
	noun := nameNouns[randomIndex(len(nameNouns))]
	return adj + "-" + noun
}

func randomIndex(max int) int {
	if max <= 0 {
		return 0
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}
	return int(binary.LittleEndian.Uint64(buf[:]) % uint64(max))
}

func nameExists(name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	root := storage.ContainersDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var cfg Config
		if err := storage.Read(storage.ContainerConfigPath(entry.Name()), &cfg); err != nil {
			continue
		}
		if cfg.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func resolveContainerID(idOrName string) (string, bool, error) {
	value := strings.TrimSpace(idOrName)
	if value == "" {
		return "", false, fmt.Errorf("empty container name")
	}
	root := storage.ContainersDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, fmt.Errorf("container %s not found", value)
		}
		return "", false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var cfg Config
		if err := storage.Read(storage.ContainerConfigPath(entry.Name()), &cfg); err != nil {
			continue
		}
		if cfg.Name == value {
			return cfg.ID, true, nil
		}
	}
	if len(value) >= 12 && isHex(value) {
		return value, false, nil
	}
	return "", false, fmt.Errorf("container %s not found", value)
}

func isHex(value string) bool {
	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch {
		case ch >= '0' && ch <= '9':
		case ch >= 'a' && ch <= 'f':
		case ch >= 'A' && ch <= 'F':
		default:
			return false
		}
	}
	return true
}

func ClampHostname(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if len(name) <= maxHostnameBytes {
		return name
	}
	return strings.TrimRight(name[:maxHostnameBytes], "-")
}
