package ipam

import (
	"log"
	"os"
	"path/filepath"

	"github.com/alexflint/go-filemutex"
)

func newFileLock(lockPath string) (*filemutex.FileMutex, error) {

	fi, err := os.Stat(lockPath)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		lockPath = filepath.Join(lockPath, "lock")
	}

	f, err := filemutex.New(lockPath)
	if err != nil {
		log.Printf("Failed in creating file lock for store: %s", err.Error())
		return nil, err
	}
	return f, nil
}
