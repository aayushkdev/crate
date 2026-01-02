package image

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ImageMetadata struct {
	Repo    string    `json:"repo"`
	Tag     string    `json:"tag"`
	Config  string    `json:"config"`
	Layers  []string  `json:"layers"`
	Created time.Time `json:"created"`
}

func imageMetaPath(ref *Reference) (string, error) {
	repo := strings.ReplaceAll(ref.Repo, "/", "_")
	fileName := repo + "-" + ref.Tag + ".json"
	root := crateRoot()

	return filepath.Join(root, "images", fileName), nil
}

func MetadataExists(ref *Reference) bool {
	path, err := imageMetaPath(ref)
	_, err = os.Stat(path)
	return err == nil
}

func WriteMetadata(ref *Reference, img *ImageManifest) error {
	path, err := imageMetaPath(ref)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	meta := ImageMetadata{
		Repo:    ref.Repo,
		Tag:     ref.Tag,
		Config:  img.Config,
		Layers:  img.Layers,
		Created: time.Now(),
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(meta)
}

func ReadMetadata(ref *Reference) (*ImageMetadata, error) {
	path, err := imageMetaPath(ref)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var meta ImageMetadata
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
