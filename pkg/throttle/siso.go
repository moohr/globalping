package throttle

// SISO (Single-Input, Single-Output) Throttles

import (
	"context"
	"fmt"
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

func (tbThrottle *TokenBasedThrottle) Run() <-chan interface{} {

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

	return tbThrottle.OutC
}

type SpeedMeasurer struct {
	RefreshInterval   time.Duration
	PreviousCounter   int
	PreviousTimestamp time.Time
	MinTimeDelta      time.Duration
}

type SpeedRecord struct {
	Timestamp        time.Time
	Counter          int
	CounterIncrement int
	TimeDelta        time.Duration
}

func (sr *SpeedRecord) String() string {
	if sr.Value() == nil {
		return "<N/A>"
	}
	return fmt.Sprintf("%f pps", *sr.Value())
}

// the unit is pps (packets per second)
func (sr *SpeedRecord) Value() *float64 {
	if sr == nil || sr.TimeDelta == 0 {
		return nil
	}

	var v float64 = float64(sr.CounterIncrement) / sr.TimeDelta.Seconds()
	return &v
}

func (sm *SpeedMeasurer) Run(inChan <-chan interface{}) (<-chan interface{}, <-chan SpeedRecord) {
	sm.PreviousCounter = 0
	sm.PreviousTimestamp = time.Now()

	outChan := make(chan interface{})
	speedRecordChan := make(chan SpeedRecord, 1)

	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	var counter *int = new(int)
	*counter = 0

	go func() {
		defer close(speedRecordChan)
		ticker := time.NewTicker(sm.RefreshInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				timeDelta := time.Since(sm.PreviousTimestamp)
				if timeDelta >= sm.MinTimeDelta {
					speedRecord := SpeedRecord{
						Timestamp:        time.Now(),
						Counter:          *counter,
						CounterIncrement: *counter - sm.PreviousCounter,
						TimeDelta:        timeDelta,
					}
					go func() {
						speedRecordChan <- speedRecord
					}()
				}
				sm.PreviousCounter = *counter
				sm.PreviousTimestamp = time.Now()
			}
		}
	}()

	go func() {
		defer close(outChan)
		defer cancel()

		for item := range inChan {
			outChan <- item
			*counter = *counter + 1
		}
	}()

	return outChan, speedRecordChan
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

func (bf *BurstSmoother) Run() <-chan interface{} {
	go func() {
		defer close(bf.OutC)
		for item := range bf.InC {
			time.Sleep(bf.LeastSampleInterval)
			bf.OutC <- item
		}
	}()

	return bf.OutC
}
