package image

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	storage "github.com/aayushkdev/crate/internal/storage"
)

type ImageMetadata struct {
	ID             string    `json:"id"`
	Repo           string    `json:"repo"`
	RepoTags       []string  `json:"repoTags"`
	ManifestDigest string    `json:"manifestDigest"`
	ConfigDigest   string    `json:"configDigest"`
	Layers         []string  `json:"layers"`
	OS             string    `json:"os"`
	Architecture   string    `json:"architecture"`
	Size           int64     `json:"size"`
	Created        time.Time `json:"created"`
}

func MetadataExists(ref *Reference) bool {
	_, err := ReadMetadata(ref)
	return err == nil
}

func WriteMetadata(ref *Reference, img *ImageManifest) error {
	meta, err := buildImageMetadata(ref, img)
	if err != nil {
		return err
	}

	existing, err := readMetadataByDigest(img.ManifestDigest)
	if err == nil {
		meta.Created = existing.Created
		meta.RepoTags = mergeRepoTags(existing.RepoTags, meta.RepoTags...)
	}

	if err := writeMetadataByDigest(meta); err != nil {
		return err
	}
	return nil
}

func buildImageMetadata(ref *Reference, img *ImageManifest) (*ImageMetadata, error) {
	layers := make([]string, 0, len(img.Layers))
	var size int64
	for _, layer := range img.Layers {
		layers = append(layers, layer.Digest)
		size += layer.Size
	}

	cfg, err := ReadImageConfig(img.ConfigDigest)
	if err != nil {
		return nil, err
	}

	osName := img.OS
	if osName == "" {
		osName = cfg.OS
	}
	arch := img.Architecture
	if arch == "" {
		arch = cfg.Architecture
	}

	return &ImageMetadata{
		ID:             shortImageID(img.ConfigDigest),
		Repo:           ref.Repo,
		RepoTags:       []string{repoTag(ref)},
		ManifestDigest: img.ManifestDigest,
		ConfigDigest:   img.ConfigDigest,
		Layers:         layers,
		OS:             osName,
		Architecture:   arch,
		Size:           size,
		Created:        time.Now(),
	}, nil
}

func writeMetadataByDigest(meta *ImageMetadata) error {
	path, err := storage.ImageMetadataPath(meta.ManifestDigest)
	if err != nil {
		return err
	}

	return storage.Write(path, meta)
}

func deleteMetadataByDigest(digest string) error {
	path, err := storage.ImageMetadataPath(digest)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func ReadMetadata(ref *Reference) (*ImageMetadata, error) {
	metas, err := readAllMetadata()
	if err != nil {
		return nil, err
	}

	tag := repoTag(ref)
	for _, meta := range metas {
		if hasRepoTag(meta, tag) {
			return meta, nil
		}
	}

	return nil, os.ErrNotExist
}

func readMetadataByDigest(digest string) (*ImageMetadata, error) {
	path, err := storage.ImageMetadataPath(digest)
	if err != nil {
		return nil, err
	}

	var meta ImageMetadata
	if err := storage.Read(path, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

func readAllMetadata() ([]*ImageMetadata, error) {
	root := storage.ImagesDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	metas := make([]*ImageMetadata, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		meta, err := readMetadataFile(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		metas = append(metas, meta)
	}

	return metas, nil
}

func readMetadataFile(path string) (*ImageMetadata, error) {
	var meta ImageMetadata
	if err := storage.Read(path, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

func RemoveRepoTag(ref *Reference, manifestDigest string) error {
	meta, err := readMetadataByDigest(manifestDigest)
	if err != nil {
		return err
	}

	tag := repoTag(ref)
	filtered := meta.RepoTags[:0]
	for _, repoTag := range meta.RepoTags {
		if repoTag != tag {
			filtered = append(filtered, repoTag)
		}
	}
	meta.RepoTags = filtered

	if len(meta.RepoTags) == 0 {
		metas, err := readAllMetadata()
		if err != nil {
			return err
		}

		if err := pruneImageBlobs(meta, metas); err != nil {
			return err
		}

		return deleteMetadataByDigest(manifestDigest)
	}

	if err := writeMetadataByDigest(meta); err != nil {
		return err
	}

	return nil
}

func repoTag(ref *Reference) string {
	return familiarRepo(ref.Repo) + ":" + ref.Tag
}

func shortImageID(digest string) string {
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return digest
	}

	hash := parts[1]
	if len(hash) > 12 {
		return hash[:12]
	}

	return hash
}

func mergeRepoTags(current []string, tags ...string) []string {
	seen := make(map[string]struct{}, len(current)+len(tags))
	merged := make([]string, 0, len(current)+len(tags))

	for _, tag := range current {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		merged = append(merged, tag)
	}

	for _, tag := range tags {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		merged = append(merged, tag)
	}

	return merged
}

func hasRepoTag(meta *ImageMetadata, tag string) bool {
	for _, repoTag := range meta.RepoTags {
		if repoTag == tag {
			return true
		}
	}

	return false
}
