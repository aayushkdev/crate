package image

import (
	"fmt"
	"os"
)

func Remove(input string) error {
	ref, err := ParseReference(input)
	if err != nil {
		return err
	}

	meta, err := ReadMetadata(ref)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("image %s not found", input)
		}
		return err
	}

	if !hasRepoTag(meta, repoTag(ref)) {
		return fmt.Errorf("image %s not found", input)
	}

	return RemoveRepoTag(ref, meta.ManifestDigest)
}

func pruneImageBlobs(target *ImageMetadata, metas []*ImageMetadata) error {
	if !blobReferencedByOtherMetadata(target.ConfigDigest, target.ManifestDigest, metas) {
		if err := deleteBlobByDigest(target.ConfigDigest); err != nil {
			return err
		}
	}

	for _, layer := range target.Layers {
		if blobReferencedByOtherMetadata(layer, target.ManifestDigest, metas) {
			continue
		}
		if err := deleteBlobByDigest(layer); err != nil {
			return err
		}
	}

	return nil
}

func blobReferencedByOtherMetadata(digest, manifestDigest string, metas []*ImageMetadata) bool {
	for _, meta := range metas {
		if meta.ManifestDigest == manifestDigest {
			continue
		}

		if meta.ConfigDigest == digest {
			return true
		}

		for _, layer := range meta.Layers {
			if layer == digest {
				return true
			}
		}
	}

	return false
}
