package container

import (
	"os"

	"github.com/aayushkdev/crate/internal/image"
	cratenet "github.com/aayushkdev/crate/internal/net"
	storage "github.com/aayushkdev/crate/internal/storage"
)

type Config struct {
	ID         string          `json:"id"`
	Name       string          `json:"name,omitempty"`
	Image      string          `json:"image"`
	Rootless   bool            `json:"rootless"`
	Cmd        []string        `json:"cmd,omitempty"`
	Env        []string        `json:"env,omitempty"`
	EntryPoint []string        `json:"entrypoint,omitempty"`
	WorkingDir string          `json:"working_dir,omitempty"`
	User       string          `json:"user,omitempty"`
	Network    cratenet.Config `json:"network,omitempty"`
}

func writeConfig(id string, meta *image.ImageMetadata, opts CreateOptions) error {
	imgCfg, err := image.ReadImageConfig(meta.ConfigDigest)
	if err != nil {
		return err
	}
	env, err := mergeEnv(imgCfg.Config.Env, opts.Env)
	if err != nil {
		return err
	}

	cfg := Config{
		ID:         id,
		Name:       opts.Name,
		Image:      primaryRepoTag(meta),
		Rootless:   os.Geteuid() != 0,
		Cmd:        imgCfg.Config.Cmd,
		Env:        env,
		EntryPoint: imgCfg.Config.Entrypoint,
		WorkingDir: imgCfg.Config.WorkingDir,
		User:       imgCfg.Config.User,
	}
	if opts.User != "" {
		cfg.User = opts.User
	}
	cfg.Network = cratenet.DefaultConfig(cfg.Rootless)
	if opts.NetworkMode != "" {
		cfg.Network.Mode = opts.NetworkMode
	}
	cfg.Network = cratenet.NormalizeConfig(cfg.Network, cfg.Rootless)
	cfg.Network.Publish = append(cfg.Network.Publish, opts.Publish...)
	var warning string
	cfg.Network, warning = cratenet.DropUnsupportedPublishedPorts(cfg.Network)
	if warning != "" {
		os.Stderr.WriteString("crate: warning: " + warning + "\n")
	}
	if err := cratenet.ValidateConfig(cfg.Network, cfg.Rootless); err != nil {
		return err
	}

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
	if resolved, ok, err := resolveContainerID(id); err != nil {
		return nil, err
	} else if ok {
		id = resolved
	}
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
