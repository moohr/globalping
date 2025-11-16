package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func handleClient(conn net.Conn, clientID int) {
	defer conn.Close()

	fmt.Printf("Client %d connected from %s\n", clientID, conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		// 读取客户端消息
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Client %d disconnected\n", clientID)
			} else {
				log.Printf("Error reading from client %d: %v\n", clientID, err)
			}
			return
		}

		fmt.Printf("Received from client %d: %s", clientID, message)

		// Echo 回客户端
		response := fmt.Sprintf("Echo (from client %d): %s", clientID, message)
		_, err = writer.WriteString(response)
		if err != nil {
			log.Printf("Error writing to client %d: %v\n", clientID, err)
			return
		}
		writer.Flush()
	}
}

func main() {
	// 清理可能存在的旧 socket 文件
	socketPath := "/var/run/myserver.sock"
	os.Remove(socketPath)

	// 创建 Unix Socket 监听器
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("Error listening:", err)
	}
	defer listener.Close()

	// 设置 socket 文件权限
	if err := os.Chmod(socketPath, 0666); err != nil {
		log.Printf("Warning: could not set socket permissions: %v", err)
	}

	fmt.Printf("Server listening on %s\n", socketPath)

	// 处理信号，优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down server...")
		listener.Close()
		os.Remove(socketPath)
		os.Exit(0)
	}()

	clientCounter := 0

	for {
		// 接受新连接
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		log.Printf("Accepted connection from %s", conn.RemoteAddr())

		clientCounter++
		// 为每个客户端启动单独的 goroutine
		go handleClient(conn, clientCounter)
	}
}
