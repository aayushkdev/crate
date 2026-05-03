package net

import (
	"fmt"
	"os"
	"os/exec"
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
	Mode          Mode   `json:"mode,omitempty"`
	Adapter       string `json:"adapter,omitempty"`
	Backend       string `json:"backend,omitempty"`
	InterfaceName string `json:"interface_name,omitempty"`
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

func ResolveRuntimeConfig(cfg Config, rootless bool) (Config, string, error) {
	cfg = NormalizeConfig(cfg, rootless)
	if err := ValidateConfig(cfg, rootless); err != nil {
		return Config{}, "", err
	}

	if !rootless || !RequiresHelper(cfg) {
		return cfg, "", nil
	}

	if _, err := exec.LookPath("pasta"); err != nil {
		fallback := cfg
		fallback.Mode = ModeNone
		fallback.Backend = ""
		fallback.InterfaceName = ""
		return fallback, "pasta not installed; continuing with networking disabled", nil
	}

	return cfg, "", nil
}

const modeEnv = "CRATE_NET_MODE"

func ModeEnv(mode Mode) string {
	return modeEnv + "=" + string(mode)
}

func ApplyModeOverride(cfg Config) Config {
	mode := Mode(os.Getenv(modeEnv))
	if mode == "" {
		return cfg
	}

	cfg.Mode = mode
	if mode != ModePrivate {
		cfg.Backend = ""
		cfg.InterfaceName = ""
	}

	return cfg
}
