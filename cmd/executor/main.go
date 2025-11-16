package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 清理可能存在的旧 socket 文件
	socketPath := "/var/run/myserver.sock"

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("Error listening:", err)
	}
	defer listener.Close()
	log.Printf("Server listening on %s\n", socketPath)

	go func() {
		muxer := http.NewServeMux()
		muxer.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello, World!"))
		})

		server := http.Server{
			Handler: muxer,
		}

		if err := server.Serve(listener); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Fatal("Error serving:", err)
			}
			log.Println("Server exitted")
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received signal: %s", sig.String())
	log.Println("\nShutting down server...")
}
