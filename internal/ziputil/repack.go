package ziputil

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RepackWithConfig unpacks srcZip, writes config.json at archive root, and creates a new zip in os.TempDir.
// The original srcZip file is not modified.
func RepackWithConfig(srcZip, configJSON string) (outZip string, release func(), err error) {
	srcInfo, err := os.Stat(srcZip)
	if err != nil {
		return "", nil, fmt.Errorf("stat source zip: %w", err)
	}
	if srcInfo.IsDir() {
		return "", nil, fmt.Errorf("source path is a directory: %s", srcZip)
	}

	tempDir, err := os.MkdirTemp("", "fbi-uploader-*")
	if err != nil {
		return "", nil, err
	}

	if err := unzip(srcZip, tempDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, err
	}

	return packWithConfig(tempDir, configJSON)
}

// PackDirWithConfig copies srcDir into a temp directory, writes config.json at archive root,
// and creates a new zip in os.TempDir. The original srcDir is not modified.
func PackDirWithConfig(srcDir, configJSON string) (outZip string, release func(), err error) {
	srcInfo, err := os.Stat(srcDir)
	if err != nil {
		return "", nil, fmt.Errorf("stat source directory: %w", err)
	}
	if !srcInfo.IsDir() {
		return "", nil, fmt.Errorf("source path is not a directory: %s", srcDir)
	}

	tempDir, err := os.MkdirTemp("", "fbi-uploader-*")
	if err != nil {
		return "", nil, err
	}

	if err := copyDir(srcDir, tempDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("copy source directory: %w", err)
	}

	return packWithConfig(tempDir, configJSON)
}

func packWithConfig(tempDir, configJSON string) (outZip string, release func(), err error) {
	removeTempDir := func() {
		_ = os.RemoveAll(tempDir)
	}

	if configJSON != "" {
		configPath := filepath.Join(tempDir, "config.json")
		if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
			removeTempDir()
			return "", nil, fmt.Errorf("write config.json: %w", err)
		}
	}

	outZip = filepath.Join(os.TempDir(), fmt.Sprintf("fbinstant-%s.zip", time.Now().UTC().Format("20060102150405")))
	if err := zipDir(tempDir, outZip); err != nil {
		removeTempDir()
		_ = os.Remove(outZip)
		return "", nil, err
	}

	release = func() {
		_ = os.Remove(outZip)
		removeTempDir()
	}
	return outZip, release, nil
}

func copyDir(src, dest string) error {
	src = filepath.Clean(src)
	dest = filepath.Clean(dest)
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		in.Close()
		out.Close()
		return copyErr
	})
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	dest = filepath.Clean(dest)
	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), dest+string(os.PathSeparator)) && filepath.Clean(target) != dest {
			return fmt.Errorf("zip slip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func zipDir(srcDir, destZip string) error {
	out, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	srcDir = filepath.Clean(srcDir)
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = rel
		hdr.Method = zip.Deflate

		writer, err := w.CreateHeader(hdr)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, f)
		f.Close()
		return copyErr
	})
}
