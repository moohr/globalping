package throttle

import (
	"context"
	"regexp"
)

type MIMOInputSourceNode struct {
	InputChan <-chan interface{}
	Label     string
	NextNode  *MIMOInputSourceNode
}

type LabeledPacketBinding struct {
	LabelPattern string
	Pattern      *regexp.Regexp
	OutputChan   chan interface{}
}

// Multiple inputs, multiple outputs scheduler
type MIMOScheduler struct {
	muxedDataChan chan interface{}
	sourceChain   *MIMOInputSourceNode
}

func NewMIMOScheduler() *MIMOScheduler {
	mimoSched := &MIMOScheduler{
		muxedDataChan: make(chan interface{}),
	}
	return mimoSched
}

// however, there's no 'DeleteInput', it would be automatically get removed once the channel is closed
func (mimoSched *MIMOScheduler) AddInput(inputChan <-chan interface{}, label string) {}

// express an interest about a specific set of packets tagged with a specific label pattern
func (mimoSched *MIMOScheduler) AddOutput(outputChan chan<- interface{}, labelPattern string, labelBindingName string) {
}

func (mimoSched *MIMOScheduler) RemoveOutput(labelBindingName string) {}

type wrappedpacket struct {
	label  string
	packet interface{}
}

func (mimoSched *MIMOScheduler) Run(ctx context.Context) {


	// muxer channel
	go func() {
		for head := mimoSched.sourceChain; head != nil; head = head.NextNode {
			select {
				case <-ctx.Done():
					return
				case packet := <-head.InputChan:
					mimoSched.muxedDataChan <- packet
					continue
				default:
					continue
			}
		}
	}()

	// demuxer channel
	go func() {
		// todo:
		// 1. pipe the muxed data to a throttle, which is in turn pipe through a smoother, which is in turn demuxed here
	}()
}
