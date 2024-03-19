package ipam

import (

	//"net/netip"

	"log"
	"net/netip"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

//"errors"
//"fmt"
//"net"

// Allocate a subnet to a tenant, populate its fields if needed
//func AllocateTenant()

//func AllocateIP()
//func NewTenantIPAM() (*TenantIPAM, error) {
//	//Create a new tenant IPAM struct
//	//Return the struct and an error if any
//	
//}

func NewNodeIPAM(store *NodeStore, nodeName string) (*NodeIPAM, error) {

	nim:= &NodeIPAM{
		NodeName: nodeName,
		NodeStore: store,
	}

	return nim, nil
}

func NewTenantIPAM(store *TenantStore, tenantName string) (*TenantIPAM, error) {

	tim:= &TenantIPAM{
		TenantName: tenantName,
		TenantStore: store,
	}
	return tim, nil
}

func ListSubnets(original string, newPrefix int) []string {

    var subnetList []string
    subnet := ipaddr.NewIPAddressString(original).GetAddress()
   
    iterator := subnet.SetPrefixLen(newPrefix).PrefixIterator()
    for iterator.HasNext() {
        subnetList = append(subnetList, iterator.Next().String())
    }
	return subnetList
}

//(nim *NodeIPAM) func AllocateTenantCIDR(subnets []string) error {


//Open node store
//Lock node store
//Get first element from tenant list
//Delete first element from tenant list
//Write to node store
//Unlock node store
//Return first element
func GetTenantIP(tenantList []string) netip.Prefix {

	subnet, err := netip.ParsePrefix(tenantList[0])
	if err != nil {
		log.Println("Error parsing tenant subnet: ", err)
	}
	return subnet
}
