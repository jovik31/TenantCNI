package ipam

import (
	"encoding/json"
	"log"
	"net"
	//"net/netip"
	"os"
	"path/filepath"
	
	//cip "github.com/containernetworking/plugins/pkg/ip"
)

const (
	defaultStoreDir   = "/var/lib/cni/tenantcni"
)


func NewTenantStore(dataDir string, tenantName string) (*TenantStore, error) {

	if dataDir == "" {
		dataDir = defaultStoreDir
	}

	dir := filepath.Join(dataDir,tenantName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	mutex, err := newFileLock(dir)
	if err != nil {
		log.Printf("Failed in creating file lock for node store: %s", err.Error())
	}
	file := filepath.Join(dir, tenantName+".json")

	
	if err != nil {
		log.Printf("Failed in parsing CIDR: %s", err.Error())
	}

	if err != nil {
		log.Printf("Failed in parsing tenant CIDR: %s", err.Error())
	}

	tenantData := &TenantData{
		IPs: make(map[string]ContainerNetInfo),
	}

	return &TenantStore{
		FileMutex: mutex,
		Directory: dir,
		Data:      tenantData,
		DataFile:  file,
	}, nil

}

func DeleteTenantStore(tenantName string) error {
	
	dataDir := defaultStoreDir

	dirpath:= filepath.Join(dataDir, tenantName)
	child, err:=os.ReadDir(dirpath)
	if err != nil {
		log.Printf("Failed reading directory %s", err)
		return err
	}
	log.Printf("Delete the following: %s", child)
	for _, entry := range child {
		err = os.RemoveAll(filepath.Join(dirpath, entry.Name()))
		log.Printf("Deleting %s", entry.Name())
		if err != nil {
			log.Printf("Failed deleting %s", err)
			return err
		}
	}
	err= os.RemoveAll(dirpath)
	if err != nil {
		log.Printf("Failed deleting %s", err)
		return err
	}
	return nil

}	

//Store tenant data to a json file
func (s *TenantStore) StoreTenantData() error {
	raw, err := json.Marshal(s.Data)
	if err != nil {
		return err
	}

	return os.WriteFile(s.DataFile, raw, 0644)
}


//Load tenant data to a tenant store
func(s *TenantStore) LoadTenantData() error {
	tenantData := &TenantData{}

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
		if err := json.Unmarshal(raw, &tenantData); err != nil {
			return err
		}
	}
	if tenantData.IPs == nil {
		tenantData.IPs = make(map[string]ContainerNetInfo)
	}

	s.Data = tenantData
	return nil
}


func(t *TenantStore) GetIPByID(id string) (net.IP, bool) {
	
	for ip, info := range t.Data.IPs {
		if info.ID == id {
			return net.ParseIP(ip), true
		}
	}
	return nil ,false
}

func (t *TenantStore) Add(ip net.IP, id string, ifname string) error{

	if len(ip) >0 {
		t.Data.IPs[ip.String()] = ContainerNetInfo{
			ID: id,
			IFname: ifname,
		}
			return t.StoreTenantData()
	}
	return nil
}



func (t *TenantStore) Contains(ip net.IP) bool{

	_, ok := t.Data.IPs[ip.String()]
	return ok

}

func (t *TenantStore) Del(id string) error {
	for ip, info := range t.Data.IPs {
		if info.ID == id {
			delete(t.Data.IPs, ip)
			log.Printf("New list %v, deleted %v", t.Data.IPs, ip)
			return t.StoreTenantData()
		}
	}
	return nil
}

func (t *TenantStore) Last() net.IP {
	return net.ParseIP(t.Data.Last)
}