package image

import (
	"encoding/json"
	"fmt"
)

type ImageManifest struct {
	Config string
	Layers []string
}

type ociIndex struct {
	Manifests []struct {
		Digest   string `json:"digest"`
		Media    string `json:"mediaType"`
		Platform struct {
			OS           string `json:"os"`
			Architecture string `json:"architecture"`
		} `json:"platform"`
	} `json:"manifests"`
}

type SingleManifest struct {
	Config struct {
		Digest string `json:"digest"`
	} `json:"config"`

	Layers []struct {
		Digest string `json:"digest"`
	} `json:"layers"`
}

func resolveManifest(ref *Reference, data []byte, contentType string) (*ImageManifest, error) {
	switch contentType {

	case "application/vnd.oci.image.index.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json":

		var idx ociIndex
		if err := json.Unmarshal(data, &idx); err != nil {
			return nil, err
		}

		for _, m := range idx.Manifests {
			// Only supporting linux and x64 for now
			if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" {
				raw, ct, err := fetchManifestByDigest(ref, m.Digest)
				if err != nil {
					return nil, err
				}
				return resolveManifest(ref, raw, ct)
			}
		}

		return nil, fmt.Errorf("no linux/amd64 image found")

	case "application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json":

		var sm SingleManifest
		if err := json.Unmarshal(data, &sm); err != nil {
			return nil, err
		}

		layers := make([]string, 0, len(sm.Layers))
		for _, l := range sm.Layers {
			layers = append(layers, l.Digest)
		}

		return &ImageManifest{
			Config: sm.Config.Digest,
			Layers: layers,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported manifest type: %s", contentType)
	}
}
