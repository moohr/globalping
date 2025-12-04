package hub

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	pkgratelimit "example.com/rbmq-demo/pkg/throttle"
)

type ServiceRequest struct {
	Func   func(ctx context.Context) error
	Result chan error
}

type ICMPTransceiveProxy interface {
	GetReader() <-chan interface{}
	GetWriter() chan<- interface{}
	Close()
}

type ICMPHubProxyEntry struct {
	// In: From Proxy user to the Hub, i.e. the proxy user writes, while the hub reads.
	In chan interface{}

	// Out: From the Hub to the Proxy user, i.e. the hub writes, while the proxy user reads.
	Out chan interface{}

	smoothedInChan chan interface{}
	labelKey       string
}

type ICMPTransceiveProxyImpl struct {
	id       int
	hubEntry *ICMPHubProxyEntry
}

func (proxy *ICMPTransceiveProxyImpl) GetReader() <-chan interface{} {
	return proxy.hubEntry.Out
}

func (proxy *ICMPTransceiveProxyImpl) GetWriter() chan<- interface{} {
	return proxy.hubEntry.In
}

func (proxy *ICMPTransceiveProxyImpl) Close() {
	close(proxy.hubEntry.In)
}

type ICMPTransceiveHub struct {
	proxies     map[int]*ICMPHubProxyEntry
	serviceChan chan chan ServiceRequest
	mimoSched   *pkgratelimit.MIMOScheduler
}

type ICMPTransceiveHubConfig struct {
	MIMOScheduler *pkgratelimit.MIMOScheduler
}

func NewICMPTransceiveHub(config *ICMPTransceiveHubConfig) *ICMPTransceiveHub {
	return &ICMPTransceiveHub{
		proxies:     make(map[int]*ICMPHubProxyEntry),
		serviceChan: make(chan chan ServiceRequest),
		mimoSched:   config.MIMOScheduler,
	}
}

func (hub *ICMPTransceiveHub) Run(ctx context.Context) {
	log.Println("[DBG] the hub is started")
	go func() {
		defer close(hub.serviceChan)
		defer log.Println("[DBG] the hub is stopped")

		for {
			requestCh := make(chan ServiceRequest)
			select {
			case <-ctx.Done():
				return
			case hub.serviceChan <- requestCh:
				request := <-requestCh
				err := request.Func(ctx)
				request.Result <- err
				close(request.Result)
			}
		}
	}()
}

func (hub *ICMPTransceiveHub) generateNextProxyID() int {
	result := -1
	maxRetries := 10
	for {
		result = rand.Intn(0xffff) & 0xffff
		if _, ok := hub.proxies[result]; !ok {
			break
		}
		maxRetries--
		if maxRetries < 0 {
			panic("failed to generate next proxy ID")
		}
	}
	return result
}

type TestPacketType string

const (
	TestPacketTypePing TestPacketType = "ping"
	TestPacketTypePong TestPacketType = "pong"
)

type TestPacket struct {
	Host string
	Type TestPacketType
	Seq  int
	Id   int
}

func (hub *ICMPTransceiveHub) GetProxy() ICMPTransceiveProxy {
	log.Println("[DBG] getting a new proxy")
	requestCh := <-hub.serviceChan
	defer close(requestCh)

	var handlerId *int = new(int)
	var hubEntry *ICMPHubProxyEntry = &ICMPHubProxyEntry{
		In:  make(chan interface{}),
		Out: make(chan interface{}),
	}

	fn := func(ctx context.Context) error {
		newId := hub.generateNextProxyID()
		hub.proxies[newId] = hubEntry
		*handlerId = newId

		return nil
	}
	request := ServiceRequest{
		Func:   fn,
		Result: make(chan error),
	}
	requestCh <- request
	<-request.Result

	hubEntry.labelKey = fmt.Sprintf("%d", *handlerId)
	err := hub.mimoSched.AddInput(hubEntry.In, hubEntry.labelKey)
	if err != nil {
		panic(fmt.Sprintf("failed to add input to mimo scheduler: %v", err))
	}

	hubEntry.smoothedInChan = make(chan interface{})
	err = hub.mimoSched.AddOutput(hubEntry.smoothedInChan, hubEntry.labelKey, hubEntry.labelKey)
	if err != nil {
		panic(fmt.Sprintf("failed to add output to mimo scheduler: %v", err))
	}

	go func() {
		for pktIn := range hubEntry.smoothedInChan {
			if testPkt, ok := pktIn.(TestPacket); ok && testPkt.Type == TestPacketTypePing {
				pong := TestPacket{
					Type: TestPacketTypePong,
					Seq:  testPkt.Seq,
					Id:   testPkt.Id,
					Host: testPkt.Host,
				}
				hubEntry.Out <- pong
				continue
			}

			// if have no idea what to do with the packet, leave it as is
			hubEntry.Out <- pktIn
		}
	}()

	return &ICMPTransceiveProxyImpl{
		id:       *handlerId,
		hubEntry: hubEntry,
	}
}
