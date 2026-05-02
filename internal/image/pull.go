package image

import "fmt"

func Pull(input string) error {
	imgRef, err := ParseReference(input)
	if err != nil {
		return err
	}

	fmt.Printf("Pulling %s from %s\n", imgRef.Tag, imgRef.Repo)

	fmt.Println("Resolving manifest")
	manifestData, contentType, manifestDigest, err := fetchManifestByTag(imgRef)
	if err != nil {
		return err
	}

	img, err := resolveManifest(imgRef, manifestData, contentType, manifestDigest)
	if err != nil {
		return err
	}

	if current, err := ReadMetadata(imgRef); err == nil && current.ManifestDigest == img.ManifestDigest {
		fmt.Println("Image already present")
		return nil
	}

	fmt.Println("Downloading blobs")

	if err := downloadBlob(imgRef, img.ConfigDigest); err != nil {
		return err
	}

	for i, layer := range img.Layers {
		fmt.Printf("Layer %d - %s \n", i+1, img.Layers[i].Digest[7:16])

		if err := downloadBlob(imgRef, layer.Digest); err != nil {
			return err
		}
	}

	oldManifestDigest := ""
	if current, err := ReadMetadata(imgRef); err == nil {
		oldManifestDigest = current.ManifestDigest
	}

	if err := WriteMetadata(imgRef, img); err != nil {
		return err
	}

	if oldManifestDigest != "" && oldManifestDigest != img.ManifestDigest {
		if err := RemoveRepoTag(imgRef, oldManifestDigest); err != nil {
			return err
		}
	}

	fmt.Println("Pull complete")
	return nil
}
