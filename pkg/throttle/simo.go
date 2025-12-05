package throttle

// SIMO (Multiple-Input, Multiple-Output) Throttles
// An SIMO remuxer/scheduler can be viewed as a combination of an MISO scheduler and a SIMO demuxer
// Here is a toy impl of fake round-robin scheduler, picking ready elements from multiple inputs
// in a not that fair manner.

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
)

// SIMOInputSourceNode represents a node in the single-linked list of input channels.
type SIMOInputSourceNode struct {
	InputChan chan interface{}
	Label     string
	NextNode  *SIMOInputSourceNode
	Dead      bool
}

// LabeledPacketBinding defines a routing rule for demultiplexing.
type LabeledPacketBinding struct {
	BindingName  string
	LabelPattern string
	Pattern      *regexp.Regexp
	OutputChan   chan<- interface{}
}

type SIMOEventType string

const (
	SIMOEVNewOutputAdded  SIMOEventType = "new_output"
	SIMOEVOutputRemoved   SIMOEventType = "output_removed"
)

type SIMOEventObject struct {
	Type SIMOEventType

	// possible payload, depends on the concrete event type
	// the requester sends this to the hub
	Payload interface{}

	// possible result, the hub sends this to the requester
	Result chan *EventResult
}

// Multiple inputs, multiple outputs scheduler
type SIMOScheduler struct {
	config *SIMOSchedConfig
	outputBindings sync.Map
	InC chan interface{}
}

type SIMOSchedConfig struct {
	ThrottleCfg *TokenBasedThrottleConfig
	SmootherCfg *BurstSmoother
	Muxer       MISOMuxer
	Demuxer     SIMODemuxer
}

func NewSIMOScheduler(config *SIMOSchedConfig) *SIMOScheduler {
	SIMOSched := &SIMOScheduler{
		config: config,
	}
	return SIMOSched
}

// AddInput adds a new labeled input channel to the source chain.
func (simoSched *SIMOScheduler) AddInput(inputChan <-chan interface{}, label string) error {
	return simoSched.config.Muxer.AddInput(inputChan, label)
}

// AddOutput expresses an interest about a specific set of packets tagged with a specific label pattern.
// Note: outputChan should be buffered if you want to allow downstream processing to be slow.
func (simoSched *SIMOScheduler) AddOutput(outputChan chan<- interface{}, labelPattern string, labelBindingName string) error {
	return simoSched.config.Demuxer.AddOutput(outputChan, labelPattern, labelBindingName)
}

// RemoveOutput removes a specific labeled output binding.
func (simoSched *SIMOScheduler) RemoveOutput(labelBindingName string) error {
	return simoSched.config.Demuxer.RemoveOutput(labelBindingName)
}

func (evObj *SIMOEventObject) handleNewOutputAdded(sched *SIMOScheduler) error {
	binding := evObj.Payload.(*LabeledPacketBinding)
	sched.outputBindings.Store(binding.BindingName, *binding)
	return nil
}

func (evObj *SIMOEventObject) handleOutputRemoved(sched *SIMOScheduler) error {
	bindingName := evObj.Payload.(string)
	sched.outputBindings.Delete(bindingName)
	return nil
}

func (simoSched *SIMOScheduler) handleEvent(evObj *SIMOEventObject) error {
	switch evObj.Type {
	case SIMOEVNewOutputAdded:
		return evObj.handleNewOutputAdded(simoSched)
	case SIMOEVOutputRemoved:
		return evObj.handleOutputRemoved(simoSched)
	default:
		panic(fmt.Sprintf("unknown event type: %s", evObj.Type))
	}
}

func (SIMOSched *SIMOScheduler) GetInput() chan<- interface{} {
	return SIMOSched.InC
}

// Run executes the scheduler, setting up the entire pipeline.
func (simoSched *SIMOScheduler) Run(ctx context.Context) {


	go func() {

		for anywrappedItem := range simoSched.InC {
			select {
			case <-ctx.Done():
				return
			default:
				wp, ok := anywrappedItem.(Wrappedpacket)
				if !ok {
					panic(fmt.Sprintf("Demuxer: Unexpected item type: %T", anywrappedItem))
				}

				// Iterate over all active output bindings
				simoSched.outputBindings.Range(func(key, value interface{}) bool {
					binding, _ := value.(LabeledPacketBinding)

					// Check if the packet's label matches the binding's pattern
					if binding.Pattern.MatchString(wp.label) {
						// It is the user's responsibility to keep the consumer side non-blocking, otherwise,
						// the scheduler will drop packets here.
						select {
						case binding.OutputChan <- wp.packet:
						default:
							log.Println("[DBG] Output channel full (client is slow). Drop the packet and continue.")
							// Output channel full (client is slow). Drop the packet and continue.
						}
					}
					// Return true to continue iterating over the bindings
					return true
				})
			}
		}
	}()

}
