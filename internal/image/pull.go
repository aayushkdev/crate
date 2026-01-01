package image

import (
	"fmt"
)

func Pull(ref string) error {

	refd, err := ParseReference(ref)
	if err != nil {
		fmt.Println(err)
	}
	data, ct, err := fetchManifestByTag(refd)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	fmt.Println(ct)

	return nil
}
