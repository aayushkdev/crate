package container

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aayushkdev/crate/internal/fs"
	"github.com/aayushkdev/crate/internal/image"
	cratenet "github.com/aayushkdev/crate/internal/net"
	storage "github.com/aayushkdev/crate/internal/storage"
)

type CreateOptions struct {
	Publish     []cratenet.PublishedPort
	NetworkMode cratenet.Mode
	Env         []string
	User        string
	Name        string
	AutoRemove  bool
	Mounts      []Mount
}

var createMu sync.Mutex

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

func Create(imageName string, opts CreateOptions) (id string, warnings []string, err error) {
	createMu.Lock()
	defer createMu.Unlock()
	ref, err := image.ParseReference(imageName)
	if err != nil {
		return "", nil, err
	}

	if !image.MetadataExists(ref) {
		fmt.Printf("Unable to find image '%s:%s' locally\n", ref.Repo, ref.Tag)
		if err := image.Pull(imageName); err != nil {
			return "", nil, err
		}
	}

	meta, err := image.ReadMetadata(ref)
	if err != nil {
		return "", nil, err
	}

	name, err := resolveContainerName(opts.Name)
	if err != nil {
		return "", nil, err
	}
	opts.Name = name

	id = generateID()
	cfg, warnings, err := buildConfig(id, meta, opts)
	if err != nil {
		return "", nil, err
	}

	cleanupID := id
	mutated := false
	defer func() {
		if err != nil && mutated {
			_ = removeContainerDir(cleanupID)
		}
	}()

	rootfs := storage.ContainerRootfsPath(id)
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return "", nil, err
	}
	mutated = true

	for _, layer := range meta.Layers {
		path, err := storage.BlobPath(layer)
		if err != nil {
			return "", nil, err
		}
		if err := fs.ApplyLayer(path, rootfs); err != nil {
			return "", nil, err
		}
	}

	if err := writeConfig(id, cfg); err != nil {
		return "", nil, err
	}

	if err := writeState(id, &State{
		ID:        id,
		Name:      name,
		Image:     familiarRef(ref),
		Status:    StatusCreated,
		LogPath:   storage.ContainerLogPath(id),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return "", nil, err
	}

	return id, warnings, nil
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
