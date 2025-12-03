package main

import (
	"fmt"
	"time"

	pkgratelimit "example.com/rbmq-demo/pkg/throttle"
)

func main() {

	N := 3000
	dataChan := make(chan interface{})
	// generator goroutine, generating mock samples at death speed
	go func() {
		defer close(dataChan)
		for i := 0; i < N; i++ {
			dataChan <- i
		}
	}()

	throttleConfig := pkgratelimit.TokenBasedThrottleConfig{
		RefreshInterval:       1 * time.Second,
		TokenQuotaPerInterval: 100,
	}
	tbThrottle := pkgratelimit.NewTokenBasedThrottle(throttleConfig)
	outChan := tbThrottle.Run(dataChan)

	speedMeasurer := pkgratelimit.SpeedMeasurer{
		RefreshInterval: 5 * time.Second,
	}

	nullChan, recorderChan := speedMeasurer.Run(outChan)
	previousCounter := 0
	previousTimestamp := time.Now()
	for {
		select {
		case <-nullChan:
			continue
		case record, ok := <-recorderChan:
			if !ok {
				break
			}
			timeDelta := record.Timestamp.Sub(previousTimestamp)
			if timeDelta.Milliseconds() <= 100 {
				continue
			}
			increment := record.Counter - previousCounter
			speed := float64(increment) / timeDelta.Seconds()
			fmt.Printf("speed: %f items/s\n", speed)
			previousCounter = record.Counter
			previousTimestamp = record.Timestamp
		}
	}

}
