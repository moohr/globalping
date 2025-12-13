package utils

import (
	"context"
	"fmt"
	"net"
)

func SelectDstIP(ctx context.Context, resolver *net.Resolver, host string, preferV4 *bool, preferV6 *bool) (*net.IPAddr, error) {
	familyPrefer := "ip"
	if preferV6 != nil && *preferV6 {
		familyPrefer = "ip6"
	} else if preferV4 != nil && *preferV4 {
		familyPrefer = "ip4"
	}
	ips, err := resolver.LookupIP(ctx, familyPrefer, host)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup IP: %v", err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP found for host: %s", host)
	}
	dst := net.IPAddr{IP: ips[0]}
	return &dst, nil
}
