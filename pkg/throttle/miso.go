package throttle

// MISO (Multiple-Input, Single-Output) muxer/scheduler

import (
	"context"
	"fmt"
)

const (
	maxNrSources int = 1000
)

// MISOInputSourceNode represents a node in the single-linked list of input channels.
type MISOInputSourceNode struct {
	InputChan chan interface{}
	Label     string
	NextNode  *MISOInputSourceNode
	Dead      bool
}

type MISOEventType string

const (
	MISOEVSourceAdded     MISOEventType = "source_added"
	MISOEVSourceDrained   MISOEventType = "source_drained"
	MISOEVNewPktAvailable MISOEventType = "new_pkt_available"
)

type MISOEventResult struct {
	Error error
}

type MISOEventObject struct {
	Type MISOEventType

	// possible payload, depends on the concrete event type
	// the requester sends this to the hub
	Payload interface{}

	// possible result, the hub sends this to the requester
	Result chan *EventResult
}

// Multiple inputs, multiple outputs scheduler
type MISOScheduler struct {

	// For interoperability, we use interface{} here instead of Wrappedpacket
	// so that it can pass nearly any type of transformer middlewares
	OutC chan interface{}

	// The head of the linked list of input sources
	sourceChain *MISOInputSourceNode

	// Throttle components configuration
	ThrottleConfig *TokenBasedThrottleConfig
	SmootherConfig *BurstSmoother

	muxerServiceChan chan chan MISOEventObject
	nrInputs         int
}

type MISOSchedConfig struct {
	ThrottleCfg *TokenBasedThrottleConfig
	SmootherCfg *BurstSmoother
}

func NewMISOScheduler(config *MISOSchedConfig) *MISOScheduler {
	MISOSched := &MISOScheduler{
		OutC:             make(chan interface{}),
		ThrottleConfig:   config.ThrottleCfg,
		SmootherConfig:   config.SmootherCfg,
		muxerServiceChan: make(chan chan MISOEventObject),
	}
	return MISOSched
}

// returns:
// (the packets collected, the node stopped)
func (sched *MISOScheduler) collectWrappedPktFromLink(head *MISOInputSourceNode) (*Wrappedpacket, *MISOInputSourceNode) {
	if head == nil {
		return nil, nil
	}

	currentNext := head

	for {
		select {
		case pkt, ok := <-currentNext.InputChan:
			if !ok {
				currentNext.Dead = true

				// go notify the hub that there is some dead source(s) need to be removed
				go func() {
					evCh, ok := <-sched.muxerServiceChan
					if !ok {
						return
					}
					defer close(evCh)
					evObj := MISOEventObject{
						Type:    MISOEVSourceDrained,
						Payload: nil,
						Result:  make(chan *EventResult),
					}
					evCh <- evObj
					<-evObj.Result
				}()
			} else {
				return &Wrappedpacket{label: currentNext.Label, packet: pkt}, currentNext.NextNode
			}
		default:
			// noop, simply for making the operation non-blocking
		}

		currentNext = currentNext.NextNode
		if currentNext == head {
			break
		}
	}
	return nil, nil
}

func insertNode(head *MISOInputSourceNode, newNode *MISOInputSourceNode) *MISOInputSourceNode {
	if head == nil {
		newNode.NextNode = newNode
		return newNode
	}

	newNode.NextNode = head.NextNode
	head.NextNode = newNode

	return head
}

func removeDeadNodes(head *MISOInputSourceNode) *MISOInputSourceNode {
	if head == nil {
		return head
	}

	var newList *MISOInputSourceNode
	originalHead := head
	for {
		if head.Dead {
			continue
		}
		newList = insertNode(newList, head)

		head = head.NextNode
		if head == originalHead {
			break
		}
	}

	return newList
}

// AddInput adds a new labeled input channel to the source chain.
func (MISOSched *MISOScheduler) AddInput(inputChan <-chan interface{}, label string) error {
	requestCh, ok := <-MISOSched.muxerServiceChan
	if !ok {
		// the engine was shutdown
		return fmt.Errorf("engine was shutdown")
	}

	defer close(requestCh)
	newNode := &MISOInputSourceNode{
		InputChan: make(chan interface{}, 1),
		Label:     label,
	}

	evObj := MISOEventObject{
		Type:    MISOEVSourceAdded,
		Payload: newNode,
		Result:  make(chan *EventResult),
	}
	requestCh <- evObj
	res := <-evObj.Result

	go func() {
		for pkt := range inputChan {
			newNode.InputChan <- pkt
			// go notify the hub that there is some new packet available
			go func() {
				evCh, ok := <-MISOSched.muxerServiceChan
				if !ok {
					// the engine was shutdown
					return
				}
				defer close(evCh)
				evObj := MISOEventObject{
					Type: MISOEVNewPktAvailable,
					// the hub will scan the whole list for any data that is available
					Payload: nil,
					Result:  make(chan *EventResult),
				}
				evCh <- evObj
				<-evObj.Result
			}()
		}
	}()

	return res.Error
}

func (evObj *MISOEventObject) handleNewInputAdded(sched *MISOScheduler) error {
	if sched.nrInputs >= maxNrSources {
		return fmt.Errorf("maximum number of inputs reached")
	}

	newNode := evObj.Payload.(*MISOInputSourceNode)
	sched.sourceChain = insertNode(sched.sourceChain, newNode)

	return nil
}

func (evObj *MISOEventObject) handleSourceDrained(sched *MISOScheduler) error {
	sched.sourceChain = removeDeadNodes(sched.sourceChain)
	return nil
}

func (evObj *MISOEventObject) handleNewPktAvailable(sched *MISOScheduler) error {
	wrappedPkt, lastNode := sched.collectWrappedPktFromLink(sched.sourceChain)
	sched.sourceChain = lastNode

	if wrappedPkt != nil {
		sched.OutC <- *wrappedPkt
	}
	sched.sourceChain = lastNode

	return nil
}

func (MISOSched *MISOScheduler) handleEvent(evObj *MISOEventObject) error {
	switch evObj.Type {
	case MISOEVSourceAdded:
		return evObj.handleNewInputAdded(MISOSched)
	case MISOEVSourceDrained:
		return evObj.handleSourceDrained(MISOSched)
	case MISOEVNewPktAvailable:
		return evObj.handleNewPktAvailable(MISOSched)
	default:
		panic(fmt.Sprintf("unknown event type: %s", evObj.Type))
	}
}

// Run executes the scheduler, setting up the entire pipeline.
func (MISOSched *MISOScheduler) Run(ctx context.Context) {

	// Runs a muxer service goroutine in the background
	go func() {
		defer close(MISOSched.muxerServiceChan)

		for {
			requestCh := make(chan MISOEventObject)
			select {
			case <-ctx.Done():
				return
			case MISOSched.muxerServiceChan <- requestCh:
				evObj := <-requestCh
				err := MISOSched.handleEvent(&evObj)
				if err != nil {
					fmt.Printf("muxer service: error handling event: %v\n", err)
				}
				evObj.Result <- &EventResult{
					Error: err,
				}
			}
		}
	}()

}
