package ipam

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"
	//filemutex "github.com/alexflint/go-filemutex"
	//cip "github.com/containernetworking/plugins/pkg/ip"
)

const (
	nodeStoreDir   = "/var/lib/cni/tenantcni"
	tenantStoreDir = "/var/lib/cni/tenantcni"
)

func NewNodeStore(dataDir string, nodeCIDR string, nodeName string, nodeIP string) (*NodeDataStore, error) {

	if dataDir == "" {
		dataDir = nodeStoreDir
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
	ipVal := net.ParseIP(nodeIP)
	_, network, err := net.ParseCIDR(nodeCIDR)
	log.Println("Network Mask: ", network.Mask)

	if err != nil {
		log.Printf("Failed in parsing CIDR: %s", err.Error())
	}

	nodeData := &NodeIPAM{
		NodeIP:       ipVal,
		NodeCIDR:     &net.IPNet{IP: network.IP, Mask: net.IPMask(network.Mask)},
		NextTenantIP: getNextTenantIP(),
		//AllowedNew: 	getMaxTenants(),
	}

	return &NodeDataStore{
		FileMutex: mutex,
		directory: dir,
		data:      nodeData,
		dataFile:  file,
	}, nil

}

// Must check every tenant network to see if it is in use or not.
// Two ways to do this: Maintain a list of all Tenant networks free to use, every time a tenant is created, delete the corresponding network from the list. When a tenant is deleted, add the network back to the list.
func getNextTenantIP() *net.IPNet {

	return &net.IPNet{IP: net.IP{192, 168, 0, 0}, Mask: net.IPMask{255, 255, 255, 0}}
}

func (s *NodeDataStore) StoreNodeData() error {

	raw, err := json.Marshal(s.data)
	if err != nil {
		return err
	}

	return os.WriteFile(s.dataFile, raw, 0644)
}
