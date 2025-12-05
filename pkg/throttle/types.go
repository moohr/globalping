package throttle

type MISOMuxer interface {
	AddInput(inputChan <-chan interface{}, label string) error
	GetOutput() <-chan interface{}
}

type SIMODemuxer interface {
	AddOutput(outputChan chan<- interface{}, labelPattern string, labelBindingName string) error
	RemoveOutput(labelBindingName string) error
	GetInput() chan<- interface{}
}

type SISOPipe interface {
	GetInput() chan<- interface{}
	GetOutput() <-chan interface{}
}

type EventResult struct {
	Error error
}

// Wrappedpacket encapsulates the original packet with its source label.
type Wrappedpacket struct {
	label  string
	packet interface{}
}
