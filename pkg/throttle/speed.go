package throttle

import (
	"context"
	"fmt"
	"time"
)

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
