package ipinfo

import (
	"context"
	"math/rand"
)

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

	randASN := randASNList[rand.Intn(len(randASNList))]
	randISP := randISPList[rand.Intn(len(randISPList))]
	randLocation := randLocationList[rand.Intn(len(randLocationList))]

	return &BasicIPInfo{
		ASN:      randASN,
		ISP:      randISP,
		Location: randLocation,
		Exact:    RandomLocation(),
	}, nil
}

func (ia *RandomIPInfoAdapter) GetName() string {
	return "random"
}
