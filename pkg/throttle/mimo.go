package throttle

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time" // Added for time.Sleep in the demuxer when no output matches
)

// --- Structs for Input Chain ---

// MIMOInputSourceNode represents a node in the single-linked list of input channels.
type MIMOInputSourceNode struct {
	InputChan <-chan interface{}
	Label     string
	NextNode  *MIMOInputSourceNode
}

// LabeledPacketBinding defines a routing rule for demultiplexing.
type LabeledPacketBinding struct {
	LabelPattern string
	Pattern      *regexp.Regexp
	OutputChan   chan<- interface{}
}

// wrappedpacket encapsulates the original packet with its source label.
type wrappedpacket struct {
	label  string
	packet interface{}
}

// --- MIMOScheduler Core Structs ---

// Multiple inputs, multiple outputs scheduler
type MIMOScheduler struct {
	// Muxed data channel (Input of the Throttle)
	muxedDataChan chan interface{} 
	// The head of the linked list of input sources
	sourceChain *MIMOInputSourceNode 
	// Mutex to protect the sourceChain when adding/removing inputs
	inputChainMux sync.Mutex 
	
	// Map of output bindings, keyed by a user-provided binding name for removal.
	// We use sync.Map because the demuxer will read from it concurrently with AddOutput/RemoveOutput.
	outputBindings sync.Map // map[string]LabeledPacketBinding

	// Throttle components configuration
	ThrottleConfig TokenBasedThrottleConfig
	SmootherConfig BurstSmoother
}

func NewMIMOScheduler(throttleCfg TokenBasedThrottleConfig, smootherCfg BurstSmoother) *MIMOScheduler {
	mimoSched := &MIMOScheduler{
		// Make this channel unbuffered to backpressure the muxer immediately if the throttle blocks.
		muxedDataChan: make(chan interface{}), 
		ThrottleConfig: throttleCfg,
		SmootherConfig: smootherCfg,
	}
	return mimoSched
}

// AddInput adds a new labeled input channel to the source chain.
func (mimoSched *MIMOScheduler) AddInput(inputChan <-chan interface{}, label string) {
	mimoSched.inputChainMux.Lock()
	defer mimoSched.inputChainMux.Unlock()

	newNode := &MIMOInputSourceNode{
		InputChan: inputChan,
		Label:     label,
		NextNode:  mimoSched.sourceChain,
	}
	// Prepend to the chain
	mimoSched.sourceChain = newNode
}

// AddOutput expresses an interest about a specific set of packets tagged with a specific label pattern.
// Note: outputChan should be buffered if you want to allow downstream processing to be slow.
func (mimoSched *MIMOScheduler) AddOutput(outputChan chan<- interface{}, labelPattern string, labelBindingName string) error {
	re, err := regexp.Compile(labelPattern)
	if err != nil {
		return fmt.Errorf("invalid label pattern %q: %w", labelPattern, err)
	}

	binding := LabeledPacketBinding{
		LabelPattern: labelPattern,
		Pattern:      re,
		OutputChan:   outputChan,
	}

	// Use Store to safely add the binding
	mimoSched.outputBindings.Store(labelBindingName, binding)
	return nil
}

// RemoveOutput removes a specific labeled output binding.
func (mimoSched *MIMOScheduler) RemoveOutput(labelBindingName string) {
	mimoSched.outputBindings.Delete(labelBindingName)
}

// Run executes the scheduler, setting up the entire pipeline.
func (mimoSched *MIMOScheduler) Run(ctx context.Context) {
	
	// --- STAGE 1: MUXER (Multiple Inputs -> Single Throttled Stream) ---
	// This goroutine constantly iterates through the input chain, multiplexing data onto muxedDataChan.
	// It relies on the channel read being non-blocking (`default: continue`) to cycle through all inputs quickly.
	go func() {
		defer close(mimoSched.muxedDataChan)
		
		fmt.Println("Muxer Goroutine started.")

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Muxer Goroutine terminated by context.")
				return
			default:
				// Use the inputChainMux to ensure thread safety while reading the chain pointer.
				mimoSched.inputChainMux.Lock() 
				current := mimoSched.sourceChain
				mimoSched.inputChainMux.Unlock()

				hasActiveInput := false
				
				for head := current; head != nil; head = head.NextNode {
					// Check if the input channel is closed or nil
					if head.InputChan == nil {
						continue 
					}
					
					// Non-blocking read from input channel
					select {
					case packet, ok := <-head.InputChan:
						if !ok {
							// Channel closed. This input is now dead.
							// In a simple linked list, removing it is tricky and slow. We'll tolerate dead nodes for simplicity here.
							fmt.Printf("Muxer: Input %s closed.\n", head.Label)
							continue 
						}
						
						// Wrap the packet with its label to enable demuxing later
						wrapped := wrappedpacket{
							label:  head.Label,
							packet: packet,
						}
						
						// Blocking send to the muxedDataChan (which is the input to the throttle).
						// This ensures the muxer backs off if the throttle/smoother pipeline is blocking.
						select {
						case mimoSched.muxedDataChan <- wrapped:
							hasActiveInput = true
						case <-ctx.Done():
							return // Check context again before blocking forever
						}

					default:
						// No packet available on this channel right now, move to next input.
					}
					hasActiveInput = true
				}

				if !hasActiveInput {
					// If all known inputs are gone or the chain is empty, take a short break to prevent 100% CPU usage.
					time.Sleep(1 * time.Millisecond)
				}
			}
		}
	}()

	// --- STAGE 2: PIPELINE (Rate Limiting and Smoothing) ---
	
	// 1. Pipe muxed data to the throttle
	throttle := NewTokenBasedThrottle(mimoSched.ThrottleConfig)
	throttledChan := throttle.Run(mimoSched.muxedDataChan) // Muxer Output -> Throttle Input
	
	// 2. Pipe throttled data to the smoother
	smoother := BurstSmoother{LeastSampleInterval: mimoSched.SmootherConfig.LeastSampleInterval}
	smoothedChan := smoother.Run(throttledChan) // Throttle Output -> Smoother Input
	
	// --- STAGE 3: DEMUXER (Single Smoothed Stream -> Multiple Outputs) ---
	
	// Demuxer goroutine reads from the fully processed channel (smoothedChan).
	go func() {
		fmt.Println("Demuxer Goroutine started.")
		
		for wrappedItem := range smoothedChan {
			select {
			case <-ctx.Done():
				fmt.Println("Demuxer Goroutine terminated by context.")
				return
			default:
				wp, ok := wrappedItem.(wrappedpacket)
				if !ok {
					fmt.Printf("Demuxer: Dropping non-wrapped item: %+v\n", wrappedItem)
					continue
				}

				// Iterate over all active output bindings
				mimoSched.outputBindings.Range(func(key, value interface{}) bool {
					binding, _ := value.(LabeledPacketBinding)
					
					// Check if the packet's label matches the binding's pattern
					if binding.Pattern.MatchString(wp.label) {
						
						// Non-blocking send to the output channel.
						// If the output channel is full (backpressure), we must drop the packet 
						// here, or the entire demuxer (and pipeline) will block.
						select {
						case binding.OutputChan <- wp.packet:
							// Packet successfully delivered.
						default:
							// Output channel full (client is slow). Drop the packet and continue.
							fmt.Printf("Demuxer: Output channel for pattern %q is full. Dropping packet from label %s.\n", binding.LabelPattern, wp.label)
						}
					}
					// Return true to continue iterating over the bindings
					return true 
				})
			}
		}
		
		fmt.Println("Demuxer Goroutine terminated because input channel closed.")
	}()

}