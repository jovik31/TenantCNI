package ipam

import (
	"net"

	store "github.com/jovik31/tenant/pkg/management/store"
)

type NodeIPAM struct {
	NodeIP        net.IP
	nodeCIDR      net.IPNet
	nextTenantIP  net.IP
	tenantsInNode int
	allowedNew    bool
	store         *store.DataStore
	tenants       []tenantIPAM
}

type tenantIPAM struct {
	tenantCIDR    net.IPNet
	bridgeName    string
	vtepName      string
	VNI           int
	bridgeGateway net.IP
	vtepIP        net.IP
	vtepMac       net.HardwareAddr
	routes        vxlanRoutes
}

type vxlanRoutes struct {
	ARP string
	FDB string
}

func newIPAM()

func getTenantCIDR()

func getNodeCIDR()

func allocateNetwork()

func releaseNetwork()
