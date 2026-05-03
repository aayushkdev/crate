package container

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/aayushkdev/crate/internal/image"
	cratenet "github.com/aayushkdev/crate/internal/net"
)

type Config struct {
	ID         string          `json:"id"`
	Image      string          `json:"image"`
	Rootless   bool            `json:"rootless"`
	Cmd        []string        `json:"cmd,omitempty"`
	Env        []string        `json:"env,omitempty"`
	EntryPoint []string        `json:"entrypoint,omitempty"`
	Network    cratenet.Config `json:"network,omitempty"`
}

func writeConfig(id string, meta *image.ImageMetadata) error {
	dir := containerDir(id)
	path := filepath.Join(dir, "config.json")

	imgCfg, err := image.ReadImageConfig(meta.ConfigDigest)
	if err != nil {
		return err
	}
	cfg := Config{
		ID:         id,
		Image:      primaryRepoTag(meta),
		Rootless:   os.Geteuid() != 0,
		Cmd:        imgCfg.Config.Cmd,
		Env:        imgCfg.Config.Env,
		EntryPoint: imgCfg.Config.Entrypoint,
	}
	cfg.Network = cratenet.DefaultConfig(cfg.Rootless)

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

func primaryRepoTag(meta *image.ImageMetadata) string {
	if len(meta.RepoTags) > 0 {
		return meta.RepoTags[0]
	}

	return meta.Repo
}

func ReadConfig(id string) (*Config, error) {
	path := filepath.Join(containerDir(id), "config.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, wrapNotFound(id, err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	cfg.Network = cratenet.NormalizeConfig(cfg.Network, cfg.Rootless)
	if err := cratenet.ValidateConfig(cfg.Network, cfg.Rootless); err != nil {
		return nil, err
	}
	return &cfg, nil
}
