package ipam

import (
	"net"

	store "github.com/jovik31/tenant/pkg/management/store"
)

type NodeIPAM struct {
	nodeIP        net.IP
	nodeCIDR      *net.IPNet
	nextTenantIP  net.IP
	allowedNew    bool
	store         *store.DataStore
	tenants       []TenantIPAM
}

type TenantIPAM struct {
	tenantCIDR		*net.IPNet
	bridge			*Bridge
	vxlan        	*Vxlan
	store           *store.DataStore
}

type Bridge struct {
	name string
	gateway net.IP
}

type Vxlan struct {
	vtepName string
	vtepIP  net.IP
	vtepMac	net.HardwareAddr
	VNI		int
}

func newIPAM()

func getTenantCIDR()

func getNodeCIDR()

func allocateNetwork()

func releaseNetwork()
