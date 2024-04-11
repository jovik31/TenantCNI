package ipam

import (
	"encoding/json"
	"log"
	"net/netip"
	"os"
	"path/filepath"
)

func NewNodeStore(dataDir string, nodeName string) (*NodeStore, error) {

	if dataDir == "" {
		dataDir = defaultStoreDir
	}

	dir := filepath.Join(dataDir, nodeName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	mutex, err := newFileLock(dir)
	if err != nil {
		log.Printf("Failed in creating file lock for node store: %s", err.Error())
	}
	file := filepath.Join(dir, nodeName+".json")

	if err != nil {
		log.Printf("Failed in parsing CIDR: %s", err.Error())
	}

	if err != nil {
		log.Printf("Failed in parsing tenant CIDR: %s", err.Error())
	}

	nodeData := &NodeData{
		AvailableList: make([]string, 0),
		TenantList:    make(map[string]netip.Prefix),
	}

	return &NodeStore{
		FileMutex: mutex,
		Directory: dir,
		Data:      nodeData,
		DataFile:  file,
	}, nil

}

func (s *NodeStore) AddAvailableTenantList(availList []string) error {
	s.Data.AvailableList = availList
	return s.StoreNodeData()
}

func (s *NodeStore) AddNodeCIDR(nodeCIDR string) error {
	
	s.Data.NodeCIDR = nodeCIDR
	return s.StoreNodeData()
}

func (s *NodeStore) AddNodeIP(nodeIP string) error {

	s.Data.NodeIP = nodeIP
	
	return s.StoreNodeData()
}

// Store node data to a json file
func (s *NodeStore) StoreNodeData() error {

	raw, err := json.Marshal(s.Data)
	if err != nil {
		return err
	}

	return os.WriteFile(s.DataFile, raw, 0644)
}

// Load node data to a node store
func (s *NodeStore) LoadNodeData() error {
	nodeData := &NodeData{}

	raw, err := os.ReadFile(s.DataFile)
	if err != nil {
		if os.IsNotExist(err) {
			f, err := os.Create(s.DataFile)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = f.Write([]byte("{}"))
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(raw, &nodeData); err != nil {
			return err
		}
	}
	if nodeData.TenantList == nil {
		nodeData.TenantList = make(map[string]netip.Prefix)
	}
	if nodeData.AvailableList == nil {
		nodeData.AvailableList = make([]string, 0)
	}

	s.Data = nodeData
	return nil
}
