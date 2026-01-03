package fs

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ApplyLayer(layerPath string, rootfs string) error {
	f, err := os.Open(layerPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(rootfs, hdr.Name)
		base := filepath.Base(hdr.Name)

		// Handle overlayfs whiteouts
		if strings.HasPrefix(base, ".wh.") {
			if base == ".wh..wh..opq" {
				dir := filepath.Dir(target)
				entries, _ := os.ReadDir(dir)
				for _, e := range entries {
					_ = os.RemoveAll(filepath.Join(dir, e.Name()))
				}
			} else {
				orig := strings.TrimPrefix(base, ".wh.")
				_ = os.RemoveAll(filepath.Join(filepath.Dir(target), orig))
			}
			continue
		}

		switch hdr.Typeflag {

		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(
				target,
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
				os.FileMode(hdr.Mode),
			)
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()

		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			linkTarget := filepath.Join(rootfs, hdr.Linkname)

			_ = os.RemoveAll(target)
			if err := os.Link(linkTarget, target); err != nil {
				return err
			}

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			_ = os.RemoveAll(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}

		default:
			// ignore other types for now
		}
	}

	return nil
}
