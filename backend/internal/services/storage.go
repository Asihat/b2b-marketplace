package services

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
)

// PublicDisk is the equivalent of Laravel's "public" filesystem disk: files
// under <root> are served by the app at <baseURL>/storage/<path>.
type PublicDisk struct {
	Root    string
	BaseURL string
}

func NewPublicDisk(storagePath, appURL string) *PublicDisk {
	return &PublicDisk{Root: filepath.Join(storagePath, "public"), BaseURL: appURL}
}

func (d *PublicDisk) fullPath(rel string) (string, error) {
	clean := filepath.Clean("/" + rel)
	if strings.Contains(clean, "..") {
		return "", errors.New("invalid storage path")
	}
	return filepath.Join(d.Root, clean), nil
}

// Put writes a file (creating directories) and returns its relative path.
func (d *PublicDisk) Put(rel string, src io.Reader) error {
	full, err := d.fullPath(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	f, err := os.Create(full)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, src)
	return err
}

func (d *PublicDisk) Delete(rel string) error {
	full, err := d.fullPath(rel)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (d *PublicDisk) Exists(rel string) bool {
	full, err := d.fullPath(rel)
	if err != nil {
		return false
	}
	_, err = os.Stat(full)
	return err == nil
}

// URL is the public URL of a stored file.
func (d *PublicDisk) URL(rel string) string {
	return httpx.AbsoluteURL(d.BaseURL, "/storage/"+strings.TrimLeft(rel, "/"))
}
