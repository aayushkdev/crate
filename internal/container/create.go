package container

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aayushkdev/crate/internal/fs"
	"github.com/aayushkdev/crate/internal/image"
	storage "github.com/aayushkdev/crate/internal/storage"
)

type CreateOptions struct {
	User string
}

func removeContainerDir(id string) error {
	return os.RemoveAll(storage.ContainerDir(id))
}

func generateID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func Create(imageName string, opts CreateOptions) (string, error) {
	ref, err := image.ParseReference(imageName)
	if err != nil {
		return "", err
	}

	if !image.MetadataExists(ref) {
		fmt.Printf("Unable to find image '%s:%s' locally\n", ref.Repo, ref.Tag)
		if err := image.Pull(imageName); err != nil {
			return "", err
		}
	}

	meta, err := image.ReadMetadata(ref)
	if err != nil {
		return "", err
	}

	id := generateID()

	rootfs := storage.ContainerRootfsPath(id)
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

	if err := writeConfig(id, meta, opts); err != nil {
		return "", err
	}

	if err := writeState(id, &State{
		ID:        id,
		Image:     familiarRef(ref),
		Status:    StatusCreated,
		LogPath:   storage.ContainerLogPath(id),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return "", err
	}

	return id, nil
}

func familiarRef(ref *image.Reference) string {
	return imageName(ref.Repo, ref.Tag)
}

func imageName(repo, tag string) string {
	if strings.HasPrefix(repo, "library/") && !strings.Contains(strings.TrimPrefix(repo, "library/"), "/") {
		repo = strings.TrimPrefix(repo, "library/")
	}

	return repo + ":" + tag
}
