package image

import "fmt"

func Pull(input string) error {

	imgRef, err := ParseReference(input)
	if err != nil {
		return err
	}

	exists := MetadataExists(imgRef)

	if exists {
		fmt.Println("Image already present")
		return nil
	}

	fmt.Printf("Pulling %s from %s\n", imgRef.Tag, imgRef.Repo)

	fmt.Println("Resolving manifest")
	manifestData, contentType, err := fetchManifestByTag(imgRef)
	if err != nil {
		return err
	}

	img, err := resolveManifest(imgRef, manifestData, contentType)
	if err != nil {
		return err
	}

	fmt.Println("Downloading blobs")

	if err := downloadBlob(imgRef, img.Config); err != nil {
		return err
	}

	for i, layer := range img.Layers {
		fmt.Printf("Layer %d - %s \n", i+1, img.Layers[i][7:16])

		if err := downloadBlob(imgRef, layer); err != nil {
			return err
		}
	}

	if err := WriteMetadata(imgRef, img); err != nil {
		return err
	}

	fmt.Println("Pull complete")
	return nil
}
