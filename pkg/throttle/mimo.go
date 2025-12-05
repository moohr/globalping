package throttle

// MIMO (Multiple-Input, Multiple-Output) Throttles
// An MIMO remuxer/scheduler can be viewed as a combination of an MISO scheduler and a SIMO demuxer, perhaps plus some middlewares.

import (
	"context"
)

// MIMOInputSourceNode represents a node in the single-linked list of input channels.
type MIMOInputSourceNode struct {
	InputChan chan interface{}
	Label     string
	NextNode  *MIMOInputSourceNode
	Dead      bool
}

// Multiple inputs, multiple outputs scheduler
type MIMOScheduler struct {
	config *MIMOSchedConfig
}

type MIMOSchedConfig struct {
	Middlewares []SISOPipe

	Muxer   MISOMuxer
	Demuxer SIMODemuxer
}

func NewMIMOScheduler(config *MIMOSchedConfig) *MIMOScheduler {
	mimoSched := &MIMOScheduler{
		config: config,
	}
	return mimoSched
}

// AddInput adds a new labeled input channel to the source chain.
func (mimoSched *MIMOScheduler) AddInput(inputChan <-chan interface{}, label string) error {
	return mimoSched.config.Muxer.AddInput(inputChan, label)
}

// AddOutput expresses an interest about a specific set of packets tagged with a specific label pattern.
// Note: outputChan should be buffered if you want to allow downstream processing to be slow.
func (mimoSched *MIMOScheduler) AddOutput(outputChan chan<- interface{}, labelPattern string, labelBindingName string) error {
	return mimoSched.config.Demuxer.AddOutput(outputChan, labelPattern, labelBindingName)
}

// RemoveOutput removes a specific labeled output binding.
func (mimoSched *MIMOScheduler) RemoveOutput(labelBindingName string) error {
	return mimoSched.config.Demuxer.RemoveOutput(labelBindingName)
}

// Run executes the scheduler, setting up the entire pipeline.
func (mimoSched *MIMOScheduler) Run(ctx context.Context) {

	var ch <-chan interface{} = mimoSched.config.Muxer.GetOutput()

	for _, middleware := range mimoSched.config.Middlewares {
		ch = WrapWithSISOPipe(ch, middleware)
	}

	nextCh := mimoSched.config.Demuxer.GetInput()

	go func() {
		for pkt := range ch {
			nextCh <- pkt
		}
	}()

}
