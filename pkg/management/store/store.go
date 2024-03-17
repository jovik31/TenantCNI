package store

import (
	"os"
	"path/filepath"

	filemutex"github.com/alexflint/go-filemutex"
	nodeConf "github.com/jovik31/tenant/pkg/ipam/types"
)

const (
	defaultDataStoreDir = "/var/lib/cni/tenants/"
)

type DataStore struct {
	*filemutex.FileMutex
	dir      string
	data     *nodeConf.NodeIPAM
	dataFile string
}

func NewNodeStore(dataDir string, nodeCIDR string) (*DataStore, error) {

	if dataDir == "" {
		dataDir = defaultDataStoreDir
	}

	dir := filepath.Join(dataDir, nodeCIDR)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	dataFile := filepath.Join(dir, nodeCIDR+".json")

	data := &nodeConf.NodeIPAM{nodeCIDR: nodeCIDR, nextTenantIP: getNextTenantIP(), allowedNew: getMaxTenants()}

	return &DataStore{dir: dir, data: data, dataFile: dataFile}, nil

}
