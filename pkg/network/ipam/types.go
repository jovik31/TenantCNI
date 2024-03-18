package ipam

import (
	"net"

	filemutex "github.com/alexflint/go-filemutex"
)

type NodeIPAM struct {
	NodeIP        net.IP `json:"nodeIP,omitempty"`
	NodeCIDR      *net.IPNet `json:"nodeCIDR,omitempty"`
	NextTenantIP  *net.IPNet `json:"nextTenantIP,omitempty"`
	//AllowedNew    bool `json:"allowedNew,omitempty"` //maybe not needed
	Tenants       *TenantData `json:"tenants,omitempty"`
}

type TenantData struct {
	IPs map[string] *net.IPNet `json:"ips"`
	LastIP string `json:"lastIP"`
}

//tenants are saved on different files due to the fact that different go routines may be changing settings.
//Since we can have seperate files than we can avoid some possible concurrent accesses to files thus allowing for faster configuration
type TenantIPAM struct {
	TenantCIDR		*net.IPNet `json:"tenantCIDR,omitempty"`
	Bridge			*Bridge `json:"bridge,omitempty"`
	Vxlan        	*Vxlan `json:"vxlan,omitempty"`
}

type Bridge struct {
	Name string `json:"name,omitempty"`
	Gateway net.IP `json:"gateway,omitempty"`
}

type Vxlan struct {
	VtepName string `json:"vtepName,omitempty"`
	VtepIP  net.IP
	VtepMac	net.HardwareAddr `json:"vtepMac,omitempty"`
	VNI		int `json:"VNI,omitempty"`
}

type containerNetInfo struct {
	ID string `json:"id"`
	IFname string `json:"ifname"`
}

type data struct {
	IPs map[string]containerNetInfo `json:"ips"`
	Last string `json:"last"`
}

type NodeDataStore struct {
	FileMutex	*filemutex.FileMutex
	directory	string		
	data     	*NodeIPAM  
	dataFile 	string
}

type TenantDataStore struct {
	FileMutex	*filemutex.FileMutex
	Directory	string
	Data     	*TenantIPAM
	DataFile 	string
}