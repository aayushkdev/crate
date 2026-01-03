package image

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func CrateRoot() string {
	var home string
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			home = u.HomeDir
		}
	}
	if home == "" {
		h, _ := os.UserHomeDir()
		home = h
	}

	return filepath.Join(home, ".local", "share", "crate")
}

func BlobPath(digest string) (string, error) {
	root := CrateRoot()
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid digest: %s", digest)
	}

	algo, hash := parts[0], parts[1]
	return filepath.Join(root, "blobs", algo, hash), nil
}

func blobExists(digest string) (bool, error) {
	path, err := BlobPath(digest)
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

	path, err := BlobPath(digest)
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
