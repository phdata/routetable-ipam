package routetable

import (
	"fmt"
	"net"
	"time"

	"github.com/TrilliumIT/iputil"
	"github.com/vishvananda/netlink"
)

// SelectAddress returns an available IP or the requested IP (if available) or an error on timeout
func SelectAddress(cidr string, xf, xl int) (*net.IPNet, error) {
	var ip *net.IPNet
	var err error
	var sleepTime time.Duration

	addr, _ := netlink.ParseIPNet(cidr)
	subn := iputil.NetworkID(addr)
	reqAddr := addr.IP
	if addr.IP.Equal(subn.IP) {
		reqAddr = nil
		sleepTime = time.Duration(DefaultRequestedAddressSleepTime) * time.Millisecond
	}

	linkIndex, _ := LinkIndexFromIPNet(addr)

	for {
		ip, err = selectAddress(reqAddr, subn, linkIndex, xf, xl)
		if err != nil {
			return nil, err
		}
		if ip != nil {
			break
		}
		time.Sleep(sleepTime)
	}

	return ip, nil
}

// selectAddress returns an available random IP on this network, or the requested IP
// if it's available. This function may return (nil, nil) if it selects an unavailable address
// the intention is for the caller to continue calling in a loop until an address is returned
// this way the caller can implement their own timeout logic
func selectAddress(reqAddress net.IP, sn *net.IPNet, linkIndex, xf, xl int) (*net.IPNet, error) {
	addrInSubnet, addrOnly := GetIPNets(reqAddress, sn)

	if reqAddress != nil && !sn.Contains(reqAddress) {
		return nil, fmt.Errorf("requested address was not in this host interface's subnet")
	}

	// keep looking for a random address until one is found
	if reqAddress == nil {
		addrOnly.IP = iputil.RandAddrWithExclude(sn, xf, xl)
		addrInSubnet.IP = addrOnly.IP
	}
	numRoutes, err := numRoutesTo(addrOnly)
	if err != nil {
		return nil, err
	}
	if numRoutes > 0 {
		return nil, nil
	}

	// add host route to routing table
	err = netlink.RouteAdd(&netlink.Route{
		LinkIndex: linkIndex,
		Dst:       addrOnly,
		Protocol:  DefaultRouteProtocol,
	})
	if err != nil {
		return nil, err
	}

	//wait for at least estimated route propagation time
	time.Sleep(time.Duration(DefaultPropagationTimeout) * time.Millisecond)

	//check that we are still the only route
	numRoutes, err = numRoutesTo(addrOnly)
	if err != nil {
		return nil, err
	}

	if numRoutes < 1 {
		// The route either wasn't successfully added, or was removed,
		// let the outer loop try again
		return nil, nil
	}

	if numRoutes == 1 {
		return addrInSubnet, nil
	}

	err = DelRoute(linkIndex, addrOnly)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

//GetIPNets takes an IP and a subnet and returns the IPNet representing the IP in the subnet,
//as well as an IPNet representing the "host only" cidr
//in other words a /32 in IPv4 or a /128 in IPv6
func GetIPNets(address net.IP, subnet *net.IPNet) (*net.IPNet, *net.IPNet) {
	sna := &net.IPNet{
		IP:   address,
		Mask: address.DefaultMask(),
	}

	//address in big subnet
	if subnet != nil {
		sna.Mask = subnet.Mask
	}

	if sna.Mask == nil {
		sna.Mask = net.CIDRMask(128, 128)
	}

	_, ml := sna.Mask.Size()
	a := &net.IPNet{
		IP:   address,
		Mask: net.CIDRMask(ml, ml),
	}

	return sna, a
}

func numRoutesTo(ipnet *net.IPNet) (int, error) {
	routes, err := netlink.RouteListFiltered(0, &netlink.Route{Dst: ipnet}, netlink.RT_FILTER_DST)
	if err != nil {
		return -1, err
	}
	return len(routes), nil
}

// DelRoute deletes the /32 or /128 to the passed address
func DelRoute(linkIndex int, ip *net.IPNet) error {
	return netlink.RouteDel(&netlink.Route{
		LinkIndex: linkIndex,
		Dst:       ip,
		Protocol:  DefaultRouteProtocol,
	})
}

//LinkIndexFromIPNet gets the link index of the first interface which is on the same subnet as the parameter
func LinkIndexFromIPNet(address *net.IPNet) (int, error) {
	routes, err := netlink.RouteGet(address.IP)
	if err != nil {
		return -1, err
	}

	for _, r := range routes {
		if r.Gw != nil {
			continue
		}

		return r.LinkIndex, nil
	}

	return -1, fmt.Errorf("interface not found")
}
