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
		defer log.Printf("[DBG] generateData %s closed", prefix)
		for i := 0; i < N; i++ {
			dataChan <- fmt.Sprintf("%s%03d", prefix, i)
			time.Sleep(50 * time.Millisecond)
		}
	}()
	return dataChan
}

func main() {
	ctx := context.Background()
	ctx, cancelSched := context.WithCancel(ctx)

	N := 50
	c1 := generateData(N, "A")
	c2 := generateData(N, "B")
	c3 := generateData(N, "C")

	throttleCfg := pkgratelimit.TokenBasedThrottleConfig{
		RefreshInterval:       1 * time.Second,
		TokenQuotaPerInterval: 5,
	}
	tbThrottle := pkgratelimit.NewTokenBasedThrottle(throttleCfg)
	tbThrottle.Run()

	smoother := pkgratelimit.NewBurstSmoother(333 * time.Millisecond)
	smoother.Run()

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
		Muxer:       tsSched,
		Middlewares: []pkgratelimit.SISOPipe{tbThrottle, smoother},
	}
	mimoSched := pkgratelimit.NewMIMOScheduler(&mimoCfg)

	defer cancelSched()
	mimoSched.Run(ctx)

	opaqueId, err := mimoSched.AddInput(ctx, c1, "A")
	if err != nil {
		log.Fatalf("failed to add input to mimo scheduler: %v", err)
	}
	log.Printf("input %v is added", opaqueId)
	opaqueId, err = mimoSched.AddInput(ctx, c2, "B")
	if err != nil {
		log.Fatalf("failed to add input to mimo scheduler: %v", err)
	}
	log.Printf("input %v is added", opaqueId)
	opaqueId, err = mimoSched.AddInput(ctx, c3, "C")
	if err != nil {
		log.Fatalf("failed to add input to mimo scheduler: %v", err)
	}
	log.Printf("input %v is added", opaqueId)

	go func() {
		// the default output of MIMO remuxer could never be closed,
		// so, run it in a dedicated goroutine, otherwise the whole thing will be deadlocked (i.e. all goroutines are asleep).
		for muxedItem := range mimoSched.DefaultOutC {
			log.Println("muxed item", muxedItem, time.Now().Format(time.RFC3339Nano))
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Println("signal received: ", sig.String(), " exitting...")
}
