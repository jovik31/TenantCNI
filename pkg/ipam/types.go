package ipam

import (
	"net"

	"github.com/jovik31/controller/pkg/store"
)

type NodeIPAM struct {
	nodeCIDR     *net.IPNet
	nextTenantIP net.IP
	allowedNew   bool
	store        *store.Store
}

type tenantIPAM struct {
	tenant  string
	subnet  *net.IPNet
	gateway net.IP //In the case of the tenant IPAM the IP address of the gateway is the one refering to the remote Vtep IP for the remote nodes
	store   *store.Store
}

func newIPAM()

func getTenantCIDR()

func getNodeCIDR()

func allocateNetwork()

func releaseNetwork()
