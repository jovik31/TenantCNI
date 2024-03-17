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
}

//tenants are saved on different files due to the fact that different go routines may be changing settings.
//Since we can have seperate files than we can avoid some possible concurrent accesses to files thus allowing for faster configuration
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
