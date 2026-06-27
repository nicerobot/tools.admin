package overrides

import (
	"path/filepath"
	"strings"

	"github.com/nicerobot/tools.admin/internal/adminerr"
)

// ReposDir is the path to a settings repos/ directory.
type ReposDir string

// OutFile is the path of a written override file.
type OutFile string

// MkdirAllFunc creates a directory tree; injected for testability.
type MkdirAllFunc func(path string) error

// WriteFileFunc writes a file's bytes; injected for testability.
type WriteFileFunc func(name string, data []byte) error

// GlobFunc lists files matching a pattern; injected for testability.
type GlobFunc func(pattern string) ([]string, error)

// RemoveFunc deletes a file; injected for testability.
type RemoveFunc func(name string) error

// Write renders f and writes it to <reposDir>/<name>.yml, creating the
// directory if needed. It returns the written path.
func Write(f File, reposDir ReposDir, mkdir MkdirAllFunc, write WriteFileFunc) (OutFile, error) {
	if err := mkdir(string(reposDir)); err != nil {
		return "", adminerr.ErrWriteFile.With(err, "dir", string(reposDir))
	}
	outfile := filepath.Join(string(reposDir), string(f.Name)+".yml")
	if err := write(outfile, []byte(f.Render())); err != nil {
		return "", adminerr.ErrWriteFile.With(err, "file", outfile)
	}
	return OutFile(outfile), nil
}

// ListExisting returns the stems (names without ".yml") of the override files
// already present in reposDir. A missing directory yields an empty list.
func ListExisting(reposDir ReposDir, glob GlobFunc) ([]string, error) {
	matches, err := glob(filepath.Join(string(reposDir), "*.yml"))
	if err != nil {
		return nil, adminerr.ErrListRepoFiles.With(err, "dir", string(reposDir))
	}
	stems := make([]string, 0, len(matches))
	for _, m := range matches {
		stems = append(stems, strings.TrimSuffix(filepath.Base(m), ".yml"))
	}
	return stems, nil
}
