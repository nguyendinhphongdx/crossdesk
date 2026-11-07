package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// safeJoin rejects archive entries that would escape dest ("zip slip").
func safeJoin(dest, name string) (string, error) {
	clean := path.Clean("/" + strings.ReplaceAll(name, `\`, "/"))
	p := filepath.Join(dest, filepath.FromSlash(clean))
	if p != dest && !strings.HasPrefix(p, dest+string(os.PathSeparator)) {
		return "", fmt.Errorf("đường dẫn không hợp lệ trong gói: %s", name)
	}
	return p, nil
}

// unzipTo extracts a zip archive into dest, keeping file modes and symlinks.
func unzipTo(archive, dest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		p, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		mode := f.Mode()
		switch {
		case mode.IsDir():
			if err := os.MkdirAll(p, 0o755); err != nil {
				return err
			}
		case mode&os.ModeSymlink != 0:
			rc, err := f.Open()
			if err != nil {
				return err
			}
			target, err := io.ReadAll(io.LimitReader(rc, 4096))
			rc.Close()
			if err != nil {
				return err
			}
			if filepath.IsAbs(string(target)) || strings.Contains(string(target), "..") {
				return fmt.Errorf("liên kết không an toàn trong gói: %s", f.Name)
			}
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(string(target), p); err != nil {
				return err
			}
		default:
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			if err := writeEntry(f, p, mode.Perm()|0o600); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeEntry(f *zip.File, p string, perm os.FileMode) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(rc, maxDownload)); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// extractFromTarGz copies the entry whose base name is want into dst.
func extractFromTarGz(archive, want, dst string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("gói không có %s", want)
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg || path.Base(h.Name) != want {
			continue
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, io.LimitReader(tr, maxDownload)); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
}

var errNotWritable = errors.New("không có quyền ghi")
