package ipinfo

import (
	"context"
	"fmt"
	"math/rand"
)

type IPInfoProviderRegistry struct {
	registeredAdapters map[string]GeneralIPInfoAdapter
}

func NewIPInfoProviderRegistry() *IPInfoProviderRegistry {
	return &IPInfoProviderRegistry{
		registeredAdapters: make(map[string]GeneralIPInfoAdapter),
	}
}

func (reg *IPInfoProviderRegistry) RegisterAdapter(adapter GeneralIPInfoAdapter) {
	reg.registeredAdapters[adapter.GetName()] = adapter
}

func (reg *IPInfoProviderRegistry) GetAdapter(name string) (GeneralIPInfoAdapter, error) {
	if adapter, ok := reg.registeredAdapters[name]; ok {
		return adapter, nil
	}
	return nil, fmt.Errorf("adapter %s not found", name)
}

type BasicIPInfo struct {
	ASN           string
	Location      string
	ISP           string
	ExactLocation []float64 // [latitude, longitude]
}

type GeneralIPInfoAdapter interface {
	GetIPInfo(ctx context.Context, ip string) (*BasicIPInfo, error)
	GetName() string
	Configure(ctx context.Context, configString string) error
}

type IPInfoAdapter struct {
	apikey *string
}

func NewIPInfoAdapter(apikey *string) GeneralIPInfoAdapter {
	return &IPInfoAdapter{
		apikey: apikey,
	}
}

func (ia *IPInfoAdapter) GetIPInfo(ctx context.Context, ip string) (*BasicIPInfo, error) {
	// todo: implement
	return nil, nil
}

func (ia *IPInfoAdapter) GetName() string {
	return "ipinfo"
}

// configString is come from ping request
func (ia *IPInfoAdapter) Configure(ctx context.Context, configString string) error {
	// todo: implement
	return nil
}

type DN42IPInfoAdapter struct{}

func NewDN42IPInfoAdapter() GeneralIPInfoAdapter {
	return &DN42IPInfoAdapter{}
}

func (ia *DN42IPInfoAdapter) GetIPInfo(ctx context.Context, ip string) (*BasicIPInfo, error) {
	// todo: implement
	return nil, nil
}

func (ia *DN42IPInfoAdapter) GetName() string {
	return "dn42"
}

func (ia *DN42IPInfoAdapter) Configure(ctx context.Context, configString string) error {
	// todo: implement
	return nil
}

type RandomIPInfoAdapter struct{}

func NewRandomIPInfoAdapter() GeneralIPInfoAdapter {
	return &RandomIPInfoAdapter{}
}

func (ia *RandomIPInfoAdapter) GetIPInfo(ctx context.Context, ip string) (*BasicIPInfo, error) {
	randASNList := []string{
		"AS65001",
		"AS65002",
		"AS65003",
		"AS65004",
		"AS65005",
		"AS65006",
	}

	randISPList := []string{
		"CT",
		"CM",
		"CU",
	}

	randLocationList := []string{
		"CN",
		"US",
		"JP",
		"KR",
		"TW",
		"HK",
		"MO",
	}

	exactLocX := rand.Float64()*180 - 90
	exactLocY := rand.Float64()*360 - 180
	exactLocation := []float64{exactLocX, exactLocY}

	randASN := randASNList[rand.Intn(len(randASNList))]
	randISP := randISPList[rand.Intn(len(randISPList))]
	randLocation := randLocationList[rand.Intn(len(randLocationList))]

	return &BasicIPInfo{
		ASN:           randASN,
		ISP:           randISP,
		Location:      randLocation,
		ExactLocation: exactLocation,
	}, nil
}

func (ia *RandomIPInfoAdapter) GetName() string {
	return "random"
}

func (ia *RandomIPInfoAdapter) Configure(ctx context.Context, configString string) error {
	return nil
}
