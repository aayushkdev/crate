package image

import (
	"fmt"
	"strings"
)

type Reference struct {
	Registry string
	Repo     string
	Tag      string
}

func ParseReference(input string) (*Reference, error) {
	if strings.HasSuffix(input, ":") {
		return nil, fmt.Errorf("empty tag is not allowed")
	}

	ref := &Reference{
		Registry: "docker.io",
		Tag:      "latest",
	}

	parts := strings.Split(input, "/")

	if len(parts) > 1 && strings.Contains(parts[0], ".") {
		ref.Registry = parts[0]
		parts = parts[1:]
	}

	repo := strings.Join(parts, "/")
	if repo == "" || strings.HasPrefix(repo, "/") || strings.HasSuffix(repo, "/") {
		return nil, fmt.Errorf("invalid image reference")
	}

	if strings.Contains(repo, ":") {
		r := strings.SplitN(repo, ":", 2)
		repo = r[0]
		ref.Tag = r[1]
	}

	if !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}

	ref.Repo = repo
	return ref, nil
}
