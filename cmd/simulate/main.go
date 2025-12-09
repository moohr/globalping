package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pkgthrottle "example.com/rbmq-demo/pkg/throttle"
)

func generateData(N int, prefix string) chan interface{} {
	dataChan := make(chan interface{})
	go func() {
		log.Printf("[DBG] generateData %s started", prefix)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	throttleConfig := pkgthrottle.TokenBasedThrottleConfig{
		RefreshInterval:       1 * time.Second,
		TokenQuotaPerInterval: 5,
	}

	tsSched, err := pkgthrottle.NewTimeSlicedEVLoopSched(&pkgthrottle.TimeSlicedEVLoopSchedConfig{})
	if err != nil {
		panic(fmt.Sprintf("failed to create time sliced event loop scheduler: %v", err))
	}
	tsSchedRunerr := tsSched.Run(ctx)

	throttle := pkgthrottle.NewTokenBasedThrottle(throttleConfig)
	throttle.Run()

	smoother := pkgthrottle.NewBurstSmoother(333 * time.Millisecond)
	smoother.Run()

	hub := pkgthrottle.NewICMPTransceiveHub(&pkgthrottle.SharedThrottleHubConfig{
		TSSched:  tsSched,
		Throttle: throttle,
		Smoother: smoother,
	})

	hub.Run(ctx)

	source1 := generateData(20, "A")
	source2 := generateData(20, "B")
	source3 := generateData(20, "C")
	source4 := generateData(20, "D")

	exitWg := sync.WaitGroup{}
	proxy1, err := hub.CreateProxy(source1)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}
	exitWg.Add(1)

	proxy2, err := hub.CreateProxy(source2)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}
	exitWg.Add(1)

	proxy3, err := hub.CreateProxy(source3)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}
	exitWg.Add(1)

	proxy4, err := hub.CreateProxy(source4)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}
	exitWg.Add(1)

	exitConsumer := make(chan interface{})
	go func() {
		for {
			select {
			case item, ok := <-proxy1:
				if !ok {
					exitWg.Done()
				}
				fmt.Println("proxy1: ", item)
			case item, ok := <-proxy2:
				if !ok {
					exitWg.Done()
				}
				fmt.Println("proxy2: ", item)
			case item, ok := <-proxy3:
				if !ok {
					exitWg.Done()
				}
				fmt.Println("proxy3: ", item)
			case item, ok := <-proxy4:
				if !ok {
					exitWg.Done()
				}
				fmt.Println("proxy4: ", item)
			case <-exitConsumer:
				return
			}
		}
	}()

	log.Println("waiting for all proxies to be closed")
	exitWg.Wait()
	close(exitConsumer)
	log.Println("all proxies are closed")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Printf("Received signal: %v, exiting...", sig)

	err = <-tsSchedRunerr
	if err != nil {
		log.Fatalf("failed to run time sliced event loop scheduler: %v", err)
	}
}
