package container

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/aayushkdev/crate/internal/fs"
	"github.com/aayushkdev/crate/internal/image"
)

func containerDir(id string) string {
	return filepath.Join(image.CrateRoot(), "containers", id)
}

func rootfsDir(id string) string {
	return filepath.Join(containerDir(id), "rootfs")
}

func generateID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func Create(imageName string) (string, error) {
	ref, err := image.ParseReference(imageName)
	if err != nil {
		return "", err
	}

	meta, err := image.ReadMetadata(ref)
	if err != nil {
		return "", err
	}

	id := generateID()

	rootfs := rootfsDir(id)
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return "", err
	}

	for _, layer := range meta.Layers {
		path, err := image.BlobPath(layer)
		if err != nil {
			return "", err
		}
		if err := fs.ApplyLayer(path, rootfs); err != nil {
			return "", err
		}
	}

	if err := writeConfig(id, meta); err != nil {
		return "", err
	}

	return id, nil
}
