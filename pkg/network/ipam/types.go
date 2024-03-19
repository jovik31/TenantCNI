package ipam

import (
	"net"
	"net/netip"

	filemutex "github.com/alexflint/go-filemutex"

)

//All network IPs that have a mask cannot be parsed onto a readable format so we saved them as strings


type NodeIPAM struct {
	NodeName string 
	NodeStore *NodeStore
}

type TenantIPAM struct {
	TenantName string
	TenantStore *TenantStore
}

type NodeData struct {
	NodeIP		netip.Addr `json:"nodeIP,omitempty"`
	NodeCIDR	netip.Prefix `json:"nodeCIDR,omitempty"`
	AvailableList	[]string `json:"availableList"`
	TenantList	map[string]netip.Prefix `json:"tenantList"`
	
}

type TenantData struct {

	TenantCIDR		netip.Prefix `json:"tenantCIDR,omitempty"`
	Bridge			*Bridge `json:"bridge,omitempty"`
	Vxlan        	*Vxlan `json:"vxlan,omitempty"`
	
	IPs map[string]ContainerNetInfo `json:"ips"`
	Last string `json:"last"`
}

type NodeStore struct {
	*filemutex.FileMutex
	Directory	string		
	Data     	*NodeData
	DataFile 	string
}


type TenantStore struct {
	*filemutex.FileMutex
	Directory	string
	Data     	*TenantData
	DataFile 	string
}

type Bridge struct {
	Name string `json:"name,omitempty"`
	Gateway netip.Addr `json:"gateway,omitempty"`
}

type Vxlan struct {
	VtepName string `json:"vtepName,omitempty"`
	VtepIP  netip.Addr `json:"vtepIP,omitempty"`
	VtepMac	net.HardwareAddr `json:"vtepMac,omitempty"`
	VNI		int `json:"VNI,omitempty"`
}

type ContainerNetInfo struct {
	ID string `json:"id"`
	IFname string `json:"ifname"`
}
