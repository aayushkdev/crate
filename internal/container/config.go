package container

import (
	"os"

	"github.com/aayushkdev/crate/internal/image"
	cratenet "github.com/aayushkdev/crate/internal/net"
	storage "github.com/aayushkdev/crate/internal/storage"
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

	if err := os.MkdirAll(storage.ContainerDir(id), 0755); err != nil {
		return err
	}

	return storage.Write(storage.ContainerConfigPath(id), &cfg)
}

func primaryRepoTag(meta *image.ImageMetadata) string {
	if len(meta.RepoTags) > 0 {
		return meta.RepoTags[0]
	}

	return meta.Repo
}

func ReadConfig(id string) (*Config, error) {
	var cfg Config
	err := storage.Read(storage.ContainerConfigPath(id), &cfg)
	if err != nil {
		return nil, wrapNotFound(id, err)
	}
	cfg.Network = cratenet.NormalizeConfig(cfg.Network, cfg.Rootless)
	if err := cratenet.ValidateConfig(cfg.Network, cfg.Rootless); err != nil {
		return nil, err
	}
	return &cfg, nil
}
