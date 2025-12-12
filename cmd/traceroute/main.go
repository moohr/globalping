package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pkgraw "example.com/rbmq-demo/pkg/raw"
)

func main() {
	// Tracing an IP packet route to www.google.com.

	const host = "www.google.com"
	ips, err := net.LookupIP(host)
	if err != nil {
		log.Fatal(err)
	}
	var dst net.IPAddr
	for _, ip := range ips {
		if ip.To4() != nil {
			dst.IP = ip
			fmt.Printf("using %v for tracing an IP packet route to %s\n", dst.IP, host)
			break
		}
	}
	if dst.IP == nil {
		log.Fatal("no A record found")
	}

	icmp4tr, err := pkgraw.NewICMP4Transceiver(pkgraw.ICMP4TransceiverConfig{
		ID:         os.Getpid() & 0xffff,
		Dst:        dst,
		PktTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create ICMP4 transceiver: %v", err)
	}

	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	if err := icmp4tr.Run(ctx); err != nil {
		log.Fatalf("failed to run ICMP4 transceiver: %v", err)
	}
	defer cancel()

	go func() {
		for reply := range icmp4tr.ReceiveC {
			log.Printf("received reply at %v: %+v", reply.ReceivedAt.Format(time.RFC3339Nano), reply)
		}
	}()

	go func() {
		pingRequests := []pkgraw.ICMPSendRequest{
			{Seq: 1, TTL: 64},
			{Seq: 2, TTL: 64},
			{Seq: 3, TTL: 64},
		}

		for _, pingRequest := range pingRequests {
			icmp4tr.SendC <- pingRequest
			log.Printf("sent ping request at %v: %+v", time.Now().Format(time.RFC3339Nano), pingRequest)
			<-time.After(1 * time.Second)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Printf("Received signal: %v, exiting...", sig.String())
}
