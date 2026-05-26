package net

import (
	"fmt"
	"strconv"
	"strings"
)

type Mode string

const (
	ModeHost    Mode = "host"
	ModeNone    Mode = "none"
	ModePrivate Mode = "private"
)

const (
	DefaultAdapter       = "crate0"
	DefaultInterfaceName = "crate0"
)

type Config struct {
	Mode          Mode            `json:"mode,omitempty"`
	Adapter       string          `json:"adapter,omitempty"`
	Backend       string          `json:"backend,omitempty"`
	InterfaceName string          `json:"interface_name,omitempty"`
	Publish       []PublishedPort `json:"publish,omitempty"`
}

type PublishedPort struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"`
}

func DefaultConfig(rootless bool) Config {
	if rootless {
		return Config{
			Mode:          ModePrivate,
			Adapter:       DefaultAdapter,
			Backend:       "pasta",
			InterfaceName: DefaultInterfaceName,
		}
	}

	return Config{
		Mode:    ModeHost,
		Adapter: DefaultAdapter,
	}
}

func NormalizeConfig(cfg Config, rootless bool) Config {
	defaults := DefaultConfig(rootless)

	if cfg.Mode == "" {
		cfg.Mode = defaults.Mode
	}
	if cfg.Adapter == "" {
		cfg.Adapter = defaults.Adapter
	}
	if cfg.Backend == "" {
		cfg.Backend = defaults.Backend
	}
	if cfg.InterfaceName == "" {
		cfg.InterfaceName = defaults.InterfaceName
	}

	return cfg
}

func ParseMode(value string) (Mode, error) {
	switch Mode(strings.TrimSpace(value)) {
	case "":
		return "", nil
	case ModeHost:
		return ModeHost, nil
	case ModeNone:
		return ModeNone, nil
	case ModePrivate:
		return ModePrivate, nil
	default:
		return "", fmt.Errorf("unsupported network mode %q (must be host, none, or private)", value)
	}
}

func DropUnsupportedPublishedPorts(cfg Config) (Config, string) {
	if len(cfg.Publish) == 0 || cfg.Mode == ModePrivate {
		return cfg, ""
	}

	mode := cfg.Mode
	if mode == "" {
		mode = "default"
	}
	cfg.Publish = nil

	return cfg, IgnoredPublishedPortsWarning(mode)
}

func IgnoredPublishedPortsWarning(mode Mode) string {
	return fmt.Sprintf("ignoring published ports because %s networking does not support port publishing", mode)
}

func ValidateConfig(cfg Config, rootless bool) error {
	switch cfg.Mode {
	case ModeHost, ModeNone:
		return nil
	case ModePrivate:
		if !rootless {
			return fmt.Errorf("private networking currently requires rootless mode")
		}
		if cfg.Adapter != DefaultAdapter {
			return fmt.Errorf("unsupported network adapter %q", cfg.Adapter)
		}
		if cfg.Backend != "pasta" {
			return fmt.Errorf("unsupported network backend %q", cfg.Backend)
		}
		if err := ValidatePublishedPorts(cfg.Publish); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported network mode %q", cfg.Mode)
	}
}

func RequiresNetNS(cfg Config) bool {
	return cfg.Mode == ModePrivate || cfg.Mode == ModeNone
}

func RequiresHelper(cfg Config) bool {
	return cfg.Mode == ModePrivate
}

func ValidatePublishedPorts(ports []PublishedPort) error {
	seen := map[string]struct{}{}
	for _, port := range ports {
		if port.HostPort < 1 || port.HostPort > 65535 {
			return fmt.Errorf("invalid host port %d", port.HostPort)
		}
		if port.ContainerPort < 1 || port.ContainerPort > 65535 {
			return fmt.Errorf("invalid container port %d", port.ContainerPort)
		}

		protocol := port.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		switch protocol {
		case "tcp", "udp":
		default:
			return fmt.Errorf("unsupported publish protocol %q", port.Protocol)
		}

		key := protocol + ":" + strconv.Itoa(port.HostPort)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate published %s port %d", protocol, port.HostPort)
		}
		seen[key] = struct{}{}
	}

	return nil
}

func PastaPortSpec(ports []PublishedPort, protocol string) string {
	specs := make([]string, 0, len(ports))
	for _, port := range ports {
		publishProtocol := port.Protocol
		if publishProtocol == "" {
			publishProtocol = "tcp"
		}
		if publishProtocol != protocol {
			continue
		}

		specs = append(specs, strconv.Itoa(port.HostPort)+":"+strconv.Itoa(port.ContainerPort))
	}

	return strings.Join(specs, ",")
}
