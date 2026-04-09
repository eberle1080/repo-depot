package depot

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const archivesDir = "archives"

// archivePath returns the destination zip path for a workspace archive.
func (d *Depot) archivePath(workspacePath string) string {
	name := filepath.Base(filepath.Clean(workspacePath))
	ts := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.zip", name, ts)

	return filepath.Join(d.root, archivesDir, filename)
}

// ArchiveAndDelete zips the workspace to depot/archives/<name>-<timestamp>.zip
// and then removes the workspace directory.
func (d *Depot) ArchiveAndDelete(workspacePath string) (string, error) {
	archiveDst := d.archivePath(workspacePath)

	if err := os.MkdirAll(filepath.Dir(archiveDst), 0o755); err != nil {
		return "", fmt.Errorf("create archives dir: %w", err)
	}

	if err := zipDir(workspacePath, archiveDst); err != nil {
		return "", fmt.Errorf("zip workspace: %w", err)
	}

	if err := os.RemoveAll(workspacePath); err != nil {
		return "", fmt.Errorf("remove workspace: %w", err)
	}

	return archiveDst, nil
}

// zipDir creates a zip archive of src at dst, preserving relative paths.
func zipDir(src, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	src = filepath.Clean(src)

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Ensure directory entries end with a slash so unzip tools recreate them.
			if rel != "." {
				_, err = w.Create(rel + "/")
			}

			return err
		}

		entry, err := w.Create(rel)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(entry, file)

		return err
	})
}
