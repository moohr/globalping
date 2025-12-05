package throttle

func WrapWithSISOPipe(origin <-chan interface{}, pipe SISOPipe) <-chan interface{} {
	outC := pipe.GetOutput()
	inC := pipe.GetInput()
	go func() {
		for pkt := range origin {
			inC <- pkt
		}
	}()
	return outC
}
