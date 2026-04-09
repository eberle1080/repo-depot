package depot

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// repoMeta records what repo-depot knows about a subtree inside a project.
type repoMeta struct {
	Prefix    string
	RemoteURL string
	ReadOnly  bool
}

// metaFilePath returns the path of the repo-depot metadata file inside a workspace.
func metaFilePath(workspacePath string) string {
	return filepath.Join(workspacePath, ".repo-depot")
}

// writeRepoMeta appends a metadata entry for a newly added subtree.
// Format (one line per repo):
//
//	<prefix> <remoteURL> <ro|rw>
func (d *Depot) writeRepoMeta(workspacePath, prefix, remoteURL string, readOnly bool) error {
	f, err := os.OpenFile(metaFilePath(workspacePath), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open meta file: %w", err)
	}
	defer f.Close()

	mode := "rw"
	if readOnly {
		mode = "ro"
	}

	if _, err := fmt.Fprintf(f, "%s %s %s\n", prefix, remoteURL, mode); err != nil {
		return fmt.Errorf("write meta entry: %w", err)
	}

	return nil
}

// readAllRepoMeta parses the metadata file and returns all known subtrees.
func (d *Depot) readAllRepoMeta(workspacePath string) ([]repoMeta, error) {
	path := metaFilePath(workspacePath)

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("open meta file: %w", err)
	}
	defer f.Close()

	var metas []repoMeta
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			return nil, fmt.Errorf("malformed meta line: %q", line)
		}

		metas = append(metas, repoMeta{
			Prefix:    parts[0],
			RemoteURL: parts[1],
			ReadOnly:  parts[2] == "ro",
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read meta file: %w", err)
	}

	return metas, nil
}
