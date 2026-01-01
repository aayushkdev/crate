package image

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func crateRoot() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "crate")
}

func blobPath(digest string) (string, error) {
	root := crateRoot()
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid digest: %s", digest)
	}

	algo, hash := parts[0], parts[1]
	return filepath.Join(root, "blobs", algo, hash), nil
}

func blobExists(digest string) (bool, error) {
	path, err := blobPath(digest)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func downloadBlob(ref *Reference, digest string) error {
	exists, err := blobExists(digest)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	token, err := fetchDockerHubToken(ref.Repo)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s/v2/%s/blobs/%s",
		dockerHubRegistry,
		ref.Repo,
		digest,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("blob download failed: %s", resp.Status)
	}

	path, err := blobPath(digest)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}
