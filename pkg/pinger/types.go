package pinger

import (
	"context"
	"encoding/json"
	"log"
	"sync"
)

type PingEvent struct {
	Data     interface{}       `json:"data,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    error             `json:"-"`
	Err      *string           `json:"err,omitempty"`
}

const MetadataKeyFrom = "from"
const MetadataKeyTarget = "target"

func (ev *PingEvent) String() string {
	j, err := json.Marshal(ev)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return ""
	}
	return string(j)
}

type Pinger interface {
	Ping(ctx context.Context) <-chan PingEvent
}

func StartMultiplePings(ctx context.Context, pingers []Pinger) <-chan PingEvent {
	eventCh := make(chan PingEvent)

	go func() {
		defer close(eventCh)

		if len(pingers) == 0 {
			return
		}

		var wg sync.WaitGroup

		for i := range pingers {
			wg.Add(1)
			go func(pinger Pinger) {
				defer wg.Done()
				for ev := range pinger.Ping(ctx) {
					eventCh <- ev
				}
			}(pingers[i])
		}

		wg.Wait()
	}()

	return eventCh
}
