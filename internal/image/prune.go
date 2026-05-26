package image

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	storage "github.com/aayushkdev/crate/internal/storage"
)

type PruneResult struct {
	MetadataRemoved int
	BlobsRemoved    int
	Reclaimed       int64
}

func Prune(stdout io.Writer) error {
	result, err := prune()
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Removed dangling image metadata: %d\n", result.MetadataRemoved)
	fmt.Fprintf(stdout, "Removed unreferenced blobs: %d\n", result.BlobsRemoved)
	fmt.Fprintf(stdout, "Reclaimed: %s\n", FormatSize(result.Reclaimed))
	return nil
}

func prune() (*PruneResult, error) {
	metas, err := readAllMetadata()
	if err != nil {
		return nil, err
	}

	result := &PruneResult{}
	referenced := make(map[string]struct{})
	for _, meta := range metas {
		if len(meta.RepoTags) == 0 {
			if err := deleteMetadataByDigest(meta.ManifestDigest); err != nil {
				return result, err
			}
			result.MetadataRemoved++
			continue
		}

		referenced[meta.ConfigDigest] = struct{}{}
		for _, layer := range meta.Layers {
			referenced[layer] = struct{}{}
		}
	}

	if err := pruneUnreferencedBlobs(referenced, result); err != nil {
		return result, err
	}

	return result, nil
}

func pruneUnreferencedBlobs(referenced map[string]struct{}, result *PruneResult) error {
	root := storage.BlobsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		algo := entry.Name()
		algoDir := filepath.Join(root, algo)
		blobs, err := os.ReadDir(algoDir)
		if err != nil {
			return err
		}
		for _, blob := range blobs {
			if blob.IsDir() {
				continue
			}

			digest := algo + ":" + blob.Name()
			if _, ok := referenced[digest]; ok {
				continue
			}

			path := filepath.Join(algoDir, blob.Name())
			info, err := blob.Info()
			if err != nil {
				return err
			}
			if err := os.Remove(path); err != nil {
				return err
			}
			result.BlobsRemoved++
			result.Reclaimed += info.Size()
		}

		_ = os.Remove(algoDir)
	}

	return nil
}
