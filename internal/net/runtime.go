package net

import (
	"os"
	"os/exec"
)

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
		fallback.Publish = nil
		warning := "pasta not installed; using none networking"
		if len(cfg.Publish) > 0 {
			warning += "; " + IgnoredPublishedPortsWarning(ModeNone)
		}
		return fallback, warning, nil
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
