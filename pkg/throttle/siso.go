package throttle

// SISO (Single-Input, Single-Output) Throttles

import (
	"context"
	"time"
)

type TokenBasedThrottleConfig struct {
	RefreshInterval       time.Duration
	TokenQuotaPerInterval int
}

type RateLimiterToken struct {
	Quota        int
	BufferedChan chan interface{}
}

type TokenBasedThrottle struct {
	InC    chan interface{}
	OutC   chan interface{}
	config TokenBasedThrottleConfig
}

func NewTokenBasedThrottle(config TokenBasedThrottleConfig) *TokenBasedThrottle {
	return &TokenBasedThrottle{
		config: config,
		InC:    make(chan interface{}),
		OutC:   make(chan interface{}),
	}
}

func (tbThrottle *TokenBasedThrottle) GetInput() chan<- interface{} {
	return tbThrottle.InC
}

func (tbThrottle *TokenBasedThrottle) GetOutput() <-chan interface{} {
	return tbThrottle.OutC
}

func (tbThrottle *TokenBasedThrottle) Run() {

	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	tokensChan := make(chan int, 1)
	tokensChan <- tbThrottle.config.TokenQuotaPerInterval

	// token generator goroutine
	go func() {
		ticker := time.NewTicker(tbThrottle.config.RefreshInterval)
		defer ticker.Stop()
		defer close(tokensChan)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tokensChan <- tbThrottle.config.TokenQuotaPerInterval
			}
		}
	}()

	// copying goroutine
	go func() {
		defer close(tbThrottle.OutC)
		defer cancel()

		quota := 0
		for item := range tbThrottle.InC {
			if quota == 0 {
				quotaInc := <-tokensChan
				quota += quotaInc
				if quota > tbThrottle.config.TokenQuotaPerInterval {
					quota = tbThrottle.config.TokenQuotaPerInterval
				}
			}
			tbThrottle.OutC <- item
			quota--
		}
	}()
}

type BurstSmoother struct {
	LeastSampleInterval time.Duration
	InC                 chan interface{}
	OutC                chan interface{}
}

func NewBurstSmoother(leastSampleInterval time.Duration) *BurstSmoother {
	return &BurstSmoother{
		LeastSampleInterval: leastSampleInterval,
		InC:                 make(chan interface{}),
		OutC:                make(chan interface{}),
	}
}

func (bf *BurstSmoother) GetInput() chan<- interface{} {
	return bf.InC
}

func (bf *BurstSmoother) GetOutput() <-chan interface{} {
	return bf.OutC
}

func (bf *BurstSmoother) Run() {
	go func() {
		defer close(bf.OutC)
		for item := range bf.InC {
			time.Sleep(bf.LeastSampleInterval)
			bf.OutC <- item
		}
	}()
}
