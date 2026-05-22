package ziputil

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepackWithConfig(t *testing.T) {
	srcDir := t.TempDir()
	srcZip := filepath.Join(srcDir, "game.zip")

	createTestZip(t, srcZip, map[string]string{
		"index.html": "<html></html>",
	})

	config := `{"backendUrl":"https://api.example.com","cdn":"/"}`
	outZip, cleanup, err := RepackWithConfig(srcZip, config)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	origData, err := os.ReadFile(srcZip)
	if err != nil {
		t.Fatal(err)
	}

	// Source zip must be unchanged (no config.json inside).
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.File {
		if f.Name == "config.json" {
			r.Close()
			t.Fatal("source zip should not contain config.json")
		}
	}
	r.Close()
	_ = origData

	r2, err := zip.OpenReader(outZip)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()

	var foundIndex, foundConfig bool
	var configContent string
	for _, f := range r2.File {
		switch f.Name {
		case "index.html":
			foundIndex = true
		case "config.json":
			foundConfig = true
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			buf := make([]byte, 256)
			n, _ := rc.Read(buf)
			configContent = string(buf[:n])
			rc.Close()
		}
	}
	if !foundIndex {
		t.Fatal("repacked zip missing index.html")
	}
	if !foundConfig {
		t.Fatal("repacked zip missing config.json")
	}
	if !strings.Contains(configContent, "backendUrl") {
		t.Fatalf("config.json content unexpected: %q", configContent)
	}

	if !strings.HasPrefix(outZip, os.TempDir()) {
		t.Fatalf("expected output under temp dir, got %s", outZip)
	}
}

func TestRepackWithoutConfig(t *testing.T) {
	srcDir := t.TempDir()
	srcZip := filepath.Join(srcDir, "game.zip")
	createTestZip(t, srcZip, map[string]string{
		"index.html": "<html></html>",
	})

	outZip, cleanup, err := RepackWithConfig(srcZip, "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	r, err := zip.OpenReader(outZip)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "config.json" {
			t.Fatal("repacked zip should not contain config.json")
		}
	}
}

func TestPackDirWithoutConfig(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	outZip, cleanup, err := PackDirWithConfig(srcDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	r, err := zip.OpenReader(outZip)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "config.json" {
			t.Fatal("packed zip should not contain config.json")
		}
	}
}

func TestPackDirWithConfig(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	config := `{"backendUrl":"https://api.example.com","cdn":"/"}`
	outZip, cleanup, err := PackDirWithConfig(srcDir, config)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	data, err := os.ReadFile(filepath.Join(srcDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "<html></html>" {
		t.Fatal("source directory should be unchanged")
	}
	if _, err := os.Stat(filepath.Join(srcDir, "config.json")); !os.IsNotExist(err) {
		t.Fatal("source directory should not contain config.json")
	}

	r, err := zip.OpenReader(outZip)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	var foundIndex, foundConfig bool
	for _, f := range r.File {
		switch f.Name {
		case "index.html":
			foundIndex = true
		case "config.json":
			foundConfig = true
		}
	}
	if !foundIndex {
		t.Fatal("packed zip missing index.html")
	}
	if !foundConfig {
		t.Fatal("packed zip missing config.json")
	}
}

func createTestZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, content := range files {
		hdr, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := hdr.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
