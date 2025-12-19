package ipinfo

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/google/btree"
)

type AutoIPInfoDispatcher struct {
	routes  *btree.BTree
	routes6 *btree.BTree
}

func NewAutoIPInfoDispatcher() *AutoIPInfoDispatcher {
	return &AutoIPInfoDispatcher{
		routes:  btree.New(2),
		routes6: btree.New(2),
	}
}

type AutoIPInfoRoute struct {
	IPNet          net.IPNet
	IPInfoProvider GeneralIPInfoAdapter
}

func (route *AutoIPInfoRoute) Less(other btree.Item) bool {
	otherRoute, ok := other.(*AutoIPInfoRoute)
	if !ok {
		panic("other is not an AutoIPInfoRoute")
	}
	ones1, _ := route.IPNet.Mask.Size()
	ones2, _ := otherRoute.IPNet.Mask.Size()

	return ones1 < ones2
}

func (autoProvider *AutoIPInfoDispatcher) SetUpDefaultRoutes(
	dn42Provider GeneralIPInfoAdapter,
	internetIPInfoProvider GeneralIPInfoAdapter,
) {
	dn42IPNet := net.IPNet{
		IP:   net.ParseIP("172.20.0.0"),
		Mask: net.CIDRMask(14, 32),
	}
	dn42IP6 := net.IPNet{
		IP:   net.ParseIP("fd00::"),
		Mask: net.CIDRMask(8, 128),
	}
	neoIPNet := net.IPNet{
		IP:   net.ParseIP("10.127.0.0"),
		Mask: net.CIDRMask(16, 32),
	}
	autoProvider.AddRoute(AutoIPInfoRoute{
		IPNet:          dn42IPNet,
		IPInfoProvider: dn42Provider,
	})
	autoProvider.AddRoute(AutoIPInfoRoute{
		IPNet:          dn42IP6,
		IPInfoProvider: dn42Provider,
	})
	autoProvider.AddRoute(AutoIPInfoRoute{
		IPNet:          neoIPNet,
		IPInfoProvider: dn42Provider,
	})
	defaultRoute := AutoIPInfoRoute{
		IPNet: net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(0, 32),
		},
		IPInfoProvider: internetIPInfoProvider,
	}
	defaultRoute6 := AutoIPInfoRoute{
		IPNet: net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
		IPInfoProvider: internetIPInfoProvider,
	}
	autoProvider.AddRoute(defaultRoute)
	autoProvider.AddRoute(defaultRoute6)
}

func (autoProvider *AutoIPInfoDispatcher) AddRoute(route AutoIPInfoRoute) {
	if route.IPNet.IP.To4() != nil {
		autoProvider.routes.ReplaceOrInsert(&route)
	} else {
		autoProvider.routes6.ReplaceOrInsert(&route)
	}
}

func (autoProvider *AutoIPInfoDispatcher) getRoute(ip net.IP) *AutoIPInfoRoute {
	var routeTable *btree.BTree = nil
	if ip.To4() != nil {
		routeTable = autoProvider.routes
	} else {
		routeTable = autoProvider.routes6
	}

	var foundRoute *AutoIPInfoRoute = new(AutoIPInfoRoute)
	var found *bool = new(bool)
	*found = false
	routeTable.Descend(func(item btree.Item) bool {
		route, ok := item.(*AutoIPInfoRoute)
		if !ok {
			panic("item is not an AutoIPInfoRoute")
		}
		if route.IPNet.Contains(ip) {
			*foundRoute = *route
			*found = true
			return false
		}
		return true
	})

	if *found {
		return foundRoute
	}
	return nil
}

func (autoProvider *AutoIPInfoDispatcher) GetIPInfo(ctx context.Context, ipAddr string) (*BasicIPInfo, error) {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip address: %s", ipAddr)
	}

	route := autoProvider.getRoute(ip)
	if route == nil {
		log.Printf("[dbg] no ipinfo provider for ip: %s", ip.String())
		return nil, nil
	}
	return route.IPInfoProvider.GetIPInfo(ctx, ip.String())
}

func (autoProvider *AutoIPInfoDispatcher) GetName() string {
	return "auto"
}
