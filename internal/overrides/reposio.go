package overrides

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nicerobot/tools.admin/internal/constants"
)

const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// ReposDir is the path to a settings repos/ directory.
type ReposDir string

// OutFile is the path of a written override file.
type OutFile string

// OSMkdir creates a directory tree with dirPerm; the production MkdirAllFunc.
func OSMkdir(path ReposDir) error { return os.MkdirAll(string(path), dirPerm) }

// OSWriteFile writes data with filePerm; the production WriteFileFunc.
func OSWriteFile(name OutFile, data []byte) error {
	return os.WriteFile(string(name), data, filePerm)
}

// MkdirAllFunc creates a directory tree; injected for testability.
type MkdirAllFunc func(path ReposDir) error

// WriteFileFunc writes a file's bytes; injected for testability.
type WriteFileFunc func(name OutFile, data []byte) error

// GlobFunc lists files matching a pattern; injected for testability.
type GlobFunc func(pattern string) ([]string, error)

// RemoveFunc deletes a file; injected for testability.
type RemoveFunc func(name string) error

// Write renders f and writes it to <reposDir>/<name>.yml, creating the
// directory if needed. It returns the written path.
func Write(f File, reposDir ReposDir, mkdir MkdirAllFunc, write WriteFileFunc) (OutFile, error) {
	if err := mkdir(reposDir); err != nil {
		return "", constants.ErrWriteFile.With(err, "dir", string(reposDir))
	}
	outfile := OutFile(filepath.Join(string(reposDir), string(f.Name)+".yml"))
	if err := write(outfile, []byte(f.Render())); err != nil {
		return "", constants.ErrWriteFile.With(err, "file", string(outfile))
	}
	return outfile, nil
}

// ListExisting returns the stems (names without ".yml") of the override files
// already present in reposDir. A missing directory yields an empty list.
func ListExisting(reposDir ReposDir, glob GlobFunc) ([]string, error) {
	matches, err := glob(filepath.Join(string(reposDir), "*.yml"))
	if err != nil {
		return nil, constants.ErrListRepoFiles.With(err, "dir", string(reposDir))
	}
	stems := make([]string, 0, len(matches))
	for _, m := range matches {
		stems = append(stems, strings.TrimSuffix(filepath.Base(m), ".yml"))
	}
	return stems, nil
}
