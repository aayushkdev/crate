package image

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const dockerHubAuthURL = "https://auth.docker.io/token"
const dockerHubRegistry = "https://registry-1.docker.io"

const manifestAccept = "" +
	"application/vnd.oci.image.manifest.v1+json, " +
	"application/vnd.docker.distribution.manifest.v2+json, " +
	"application/vnd.oci.image.index.v1+json, " +
	"application/vnd.docker.distribution.manifest.list.v2+json"

type tokenResponse struct {
	Token string `json:"token"`
}

func fetchDockerHubToken(repo string) (string, error) {
	resp, err := http.Get(
		fmt.Sprintf(
			"%s?service=registry.docker.io&scope=repository:%s:pull",
			dockerHubAuthURL, repo,
		),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth failed: %s", resp.Status)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	if tr.Token == "" {
		return "", fmt.Errorf("empty auth token")
	}

	return tr.Token, nil
}

func fetchManifest(ref *Reference, name string) ([]byte, string, error) {
	if ref.Registry != "docker.io" {
		return nil, "", fmt.Errorf("only docker.io supported")
	}

	token, err := fetchDockerHubToken(ref.Repo)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/v2/%s/manifests/%s", dockerHubRegistry, ref.Repo, name),
		nil,
	)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Accept", manifestAccept)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("manifest fetch failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return body, resp.Header.Get("Content-Type"), nil
}

func fetchManifestByTag(ref *Reference) ([]byte, string, error) {
	return fetchManifest(ref, ref.Tag)
}

func fetchManifestByDigest(ref *Reference, digest string) ([]byte, string, error) {
	return fetchManifest(ref, digest)
}
