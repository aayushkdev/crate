package container

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/aayushkdev/crate/internal/image"
)

type Config struct {
	ID         string   `json:"id"`
	Image      string   `json:"image"`
	Rootless   bool     `json:"rootless"`
	Cmd        []string `json:"cmd,omitempty"`
	Env        []string `json:"env,omitempty"`
	EntryPoint []string `json:"entrypoint,omitempty"`
}

func writeConfig(id string, meta *image.ImageMetadata) error {
	dir := containerDir(id)
	path := filepath.Join(dir, "config.json")

	imgCfg, err := image.ReadImageConfig(meta.Config)
	if err != nil {
		return err
	}
	cfg := Config{
		ID:         id,
		Image:      meta.Repo + ":" + meta.Tag,
		Rootless:   os.Geteuid() != 0,
		Cmd:        imgCfg.Config.Cmd,
		Env:        imgCfg.Config.Env,
		EntryPoint: imgCfg.Config.Entrypoint,
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

func ReadConfig(id string) (*Config, error) {
	path := filepath.Join(containerDir(id), "config.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
