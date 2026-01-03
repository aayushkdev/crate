package image

import (
	"encoding/json"
	"os"
)

type ImageConfig struct {
	Config struct {
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

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg ImageConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
