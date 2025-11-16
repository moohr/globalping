package utils

import "net/http"

func TryFlush(w http.ResponseWriter) {
	// The default HTTP/1.x and HTTP/2 ResponseWriter implementations support Flusher,
	// but ResponseWriter wrappers may not.
	// Handlers should always test for this ability at runtime.
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
