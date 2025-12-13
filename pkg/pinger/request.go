package pinger

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
)

type SimplePingRequest struct {
	ICMPId                     *int
	Destination                string
	IntvMilliseconds           int
	PktTimeoutMilliseconds     int
	PreferV4                   *bool
	PreferV6                   *bool
	TotalPkts                  *int
	Resolver                   *string
	TTL                        *int
	ResolveTimeoutMilliseconds *int
}

const ParamCount = "count"
const paramIntvMs = "intervalMs"
const paramPktTimeoutMs = "pktTimeoutMs"
const paramTTL = "ttl"
const paramPreferV4 = "preferV4"
const paramPreferV6 = "preferV6"
const paramResolver = "resolver"
const paramICMPId = "id"
const paramDestination = "destination"
const paramResolveTimeoutMilliseconds = "resolveTimeoutMilliseconds"

func ParseSimplePingRequest(r *http.Request) (*SimplePingRequest, error) {
	result := new(SimplePingRequest)
	if count := r.URL.Query().Get(ParamCount); count != "" {
		countInt, err := strconv.Atoi(count)
		if err != nil {
			return nil, fmt.Errorf("failed to parse count: %v", err)
		}
		result.TotalPkts = &countInt
	}

	if intervalMilliSecs := r.URL.Query().Get(paramIntvMs); intervalMilliSecs != "" {
		intervalInt, err := strconv.Atoi(intervalMilliSecs)
		if err != nil {
			return nil, fmt.Errorf("failed to parse interval: %v", err)
		}
		result.IntvMilliseconds = intervalInt
	} else {
		result.IntvMilliseconds = 1000
	}

	if pktTimeoutMilliSecs := r.URL.Query().Get(paramPktTimeoutMs); pktTimeoutMilliSecs != "" {
		pktTimeoutInt, err := strconv.Atoi(pktTimeoutMilliSecs)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pktTimeout: %v", err)
		}
		result.PktTimeoutMilliseconds = pktTimeoutInt
	} else {
		result.PktTimeoutMilliseconds = 3000
	}

	if ttl := r.URL.Query().Get(paramTTL); ttl != "" {
		ttlInt, err := strconv.Atoi(ttl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ttl: %v", err)
		}
		result.TTL = &ttlInt
	}

	if preferV4 := r.URL.Query().Get(paramPreferV4); preferV4 != "" {
		preferV4Bool, err := strconv.ParseBool(preferV4)
		if err != nil {
			return nil, fmt.Errorf("failed to parse preferV4: %v", err)
		}
		result.PreferV4 = &preferV4Bool
	}

	if preferV6 := r.URL.Query().Get(paramPreferV6); preferV6 != "" {
		preferV6Bool, err := strconv.ParseBool(preferV6)
		if err != nil {
			return nil, fmt.Errorf("failed to parse preferV6: %v", err)
		}
		result.PreferV6 = &preferV6Bool
	}

	if resolver := r.URL.Query().Get(paramResolver); resolver != "" {
		result.Resolver = &resolver
	}

	if icmpId := r.URL.Query().Get(paramICMPId); icmpId != "" {
		idInt, err := strconv.Atoi(icmpId)
		if err != nil {
			return nil, fmt.Errorf("failed to parse id: %v", err)
		}
		result.ICMPId = &idInt
	} else {
		idInt := rand.Intn(0x10000)
		result.ICMPId = &idInt
	}

	destination := r.URL.Query().Get(paramDestination)
	if destination == "" {
		return nil, fmt.Errorf("destination is required")
	}
	result.Destination = destination

	return result, nil
}

func (pr *SimplePingRequest) ToURLValues() url.Values {
	vals := url.Values{}

	if pr.ICMPId != nil {
		vals.Add(paramICMPId, strconv.Itoa(*pr.ICMPId))
	}

	vals.Add(paramDestination, pr.Destination)
	vals.Add(paramIntvMs, strconv.Itoa(pr.IntvMilliseconds))
	vals.Add(paramPktTimeoutMs, strconv.Itoa(pr.PktTimeoutMilliseconds))
	if pr.PreferV4 != nil {
		vals.Add(paramPreferV4, strconv.FormatBool(*pr.PreferV4))
	}
	if pr.PreferV6 != nil {
		vals.Add(paramPreferV6, strconv.FormatBool(*pr.PreferV6))
	}
	if pr.TotalPkts != nil {
		vals.Add(ParamCount, strconv.Itoa(*pr.TotalPkts))
	}
	if pr.Resolver != nil {
		vals.Add(paramResolver, *pr.Resolver)
	}
	if pr.TTL != nil {
		vals.Add(paramTTL, strconv.Itoa(*pr.TTL))
	}
	if pr.ResolveTimeoutMilliseconds != nil {
		vals.Add(paramResolveTimeoutMilliseconds, strconv.Itoa(*pr.ResolveTimeoutMilliseconds))
	}

	return vals
}
