package image

import (
	storage "github.com/aayushkdev/crate/internal/storage"
)

type ImageConfig struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Config       struct {
		Cmd        []string `json:"Cmd"`
		Env        []string `json:"Env"`
		WorkingDir string   `json:"WorkingDir"`
		User       string   `json:"User"`
		Entrypoint []string `json:"Entrypoint"`
	} `json:"config"`
}

func ReadImageConfig(digest string) (*ImageConfig, error) {
	path, err := BlobPath(digest)
	if err != nil {
		return nil, err
	}

	var cfg ImageConfig
	if err := storage.Read(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
