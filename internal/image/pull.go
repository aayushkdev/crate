package image

import "fmt"

func Pull(input string) error {

	imgRef, err := ParseReference(input)
	if err != nil {
		return err
	}

	fmt.Printf("Pulling %s/%s:%s\n", imgRef.Registry, imgRef.Repo, imgRef.Tag)

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
		fmt.Printf("Layer %d - %s \n", i+1, img.Layers[i])

		if err := downloadBlob(imgRef, layer); err != nil {
			return err
		}
	}

	fmt.Println("Pull complete")
	return nil
}
