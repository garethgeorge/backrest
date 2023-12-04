package fsutil

import (
	"fmt"
	"os"
	"path"
)

// CollapseSubdirectory recursively moves the subdir up towards parent and removes any (newly empty) directories on the way.
func CollapseSubdirectory(p, subdir string) (string, error) {
	if p == subdir {
		return p, nil
	}

	dir := path.Dir(subdir)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read dir %q: %w", dir, err)
	}

	base := path.Base(subdir)
	for _, ent := range ents {
		if ent.Name() != base && ent.Name() != "." && ent.Name() != ".." {
			return subdir, nil
		}
	}

	if err := os.Rename(subdir, dir+".tmp"); err != nil {
		return "", fmt.Errorf("rename %q to %q: %w", subdir, dir+".tmp", err)
	}

	if err := os.Remove(dir); err != nil {
		return "", fmt.Errorf("remove %q: %w", dir, err)
	}

	if err := os.Rename(dir+".tmp", base); err != nil {
		return "", fmt.Errorf("rename %q to %q: %w", dir+".tmp", base, err)
	}

	return CollapseSubdirectory(p, dir)
}
