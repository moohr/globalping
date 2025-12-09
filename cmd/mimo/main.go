package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	pkgratelimit "example.com/rbmq-demo/pkg/throttle"
)

func generateData(N int, prefix string) chan interface{} {
	dataChan := make(chan interface{})
	go func() {
		defer close(dataChan)
		for i := 0; i < N; i++ {
			fmt.Printf("[DBG] Generate %d th item for %s\n", i, prefix)
			dataChan <- fmt.Sprintf("%s%03d", prefix, i)
		}
	}()
	return dataChan
}

func main() {
	ctx := context.Background()
	ctx, cancelSched := context.WithCancel(ctx)

	N := 100
	c1 := generateData(N, "A")
	c2 := generateData(N, "B")
	c3 := generateData(N, "C")

	// throttleCfg := pkgratelimit.TokenBasedThrottleConfig{
	// 	RefreshInterval:       1 * time.Second,
	// 	TokenQuotaPerInterval: 10,
	// }
	// throttle := pkgratelimit.NewTokenBasedThrottle(throttleCfg)
	// burstSmoother := pkgratelimit.NewBurstSmoother(100 * time.Millisecond)
	tsSched, err := pkgratelimit.NewTimeSlicedEVLoopSched(&pkgratelimit.TimeSlicedEVLoopSchedConfig{})
	go func() {
		err := <-tsSched.Run(ctx)
		if err != nil {
			log.Fatalf("failed to run time sliced event loop scheduler: %v", err)
		}
	}()

	if err != nil {
		log.Fatalf("failed to create time sliced event loop scheduler: %v", err)
	}
	mimoCfg := pkgratelimit.MIMOSchedConfig{

		Muxer: tsSched,
	}

	mimoSched := pkgratelimit.NewMIMOScheduler(&mimoCfg)

	defer cancelSched()
	mimoSched.Run(ctx)

	go func() {
		mimoSched.AddInput(ctx, c1, "A")
		mimoSched.AddInput(ctx, c2, "B")
		mimoSched.AddInput(ctx, c3, "C")
	}()

	for muxedItem := range mimoSched.DefaultOutC {
		log.Println("muxed item", muxedItem, time.Now().Format(time.RFC3339Nano))
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Println("signal received: ", sig.String(), " exitting...")
}
