package image

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
