package store

import (
	"os"
	"path/filepath"

	"github.com/alexflint/go-filemutex"
)

func newFileLock(path string) (*filemutex.FileMutex, error) {

	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		lockPath := filepath.Join(path, "lock")
	}
	return filemutex.New(path)
}
