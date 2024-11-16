package ipam

import (
	"net/netip"

	filemutex "github.com/alexflint/go-filemutex"
)

//All network IPs that have a mask cannot be parsed onto a readable format so we saved them as strings

type NodeIPAM struct {
	NodeName  string
	NodeStore *NodeStore
}

type TenantIPAM struct {
	TenantName  string
	TenantStore *TenantStore
}

type PodIPAM struct {
	PodStore *PodStore
}

type NodeData struct {
	NodeIP        string                  `json:"nodeIP"`
	NodeCIDR      string                  `json:"nodeCIDR"`
	AvailableList []string                `json:"availableList"`
	TenantList    map[string]netip.Prefix `json:"tenantList"`
}

type TenantData struct {
	TenantName   string  `json:"tenantName"`
	TenantPrefix int     `json:"tenantPrefix"`
	TenantCIDR   string  `json:"tenantCIDR"`
	Bridge       *Bridge `json:"bridge"`
	Vxlan        *Vxlan  `json:"vxlan"`

	IPs  map[string]ContainerNetInfo `json:"ips"`
	Last string                      `json:"last"`
}

type PodData struct {
	Pods map[string]string `json:"pods"`
}

type NodeStore struct {
	*filemutex.FileMutex
	Directory string
	Data      *NodeData
	DataFile  string
}

type TenantStore struct {
	*filemutex.FileMutex
	Directory string
	Data      *TenantData
	DataFile  string
}

type PodStore struct {
	*filemutex.FileMutex
	Directory string
	Data      *PodData
	DataFile  string
}

type Bridge struct {
	Name    string     `json:"name"`
	Gateway netip.Addr `json:"gateway"`
}

type Vxlan struct {
	VtepName string `json:"vtepName"`
	VtepIP   string `json:"vtepIP"`
	VtepMac  string `json:"vtepMac"`
	VNI      int    `json:"VNI"`
}

type ContainerNetInfo struct {
	ID     string `json:"id"`
	IFname string `json:"ifname"`
	NetNS  string `json: netns`
	Name   string `json: name`
}
