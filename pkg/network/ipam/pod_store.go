package ipam

import (
	"encoding/json"
	"log"
	//"net/netip"
	"os"
	"path/filepath"
	
	//cip "github.com/containernetworking/plugins/pkg/ip"
)

const (
	defaultPodStore= "/var/lib/cni/tenantcni/podlist"
)


func NewPodStore()(*PodStore, error) {


	dir := filepath.Join(defaultPodStore)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	mutex, err := newFileLock(dir)
	if err != nil {
		log.Printf("Failed in creating file lock for pod store: %s", err.Error())
	}
	file := filepath.Join(dir, "podlist.json")

	podData:= &PodData{
		Pods: make(map[string]string),
	}

	return &PodStore{
		FileMutex: mutex,
		Directory: dir,
		Data:      podData,
		DataFile:  file,
	}, nil

}

func (s *PodStore) StorePodData() error {
	raw, err := json.Marshal(s.Data)
	if err != nil {
		return err
	}

	return os.WriteFile(s.DataFile, raw, 0644)
}


func(s *PodStore) LoadPodData() error {
	podData := &PodData{}

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
		if err := json.Unmarshal(raw, &podData); err != nil {
			return err
		}
	}
	if podData.Pods == nil {
		podData.Pods = make(map[string]string)
	}

	s.Data = podData
	return nil
}
