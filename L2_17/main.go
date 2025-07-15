package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"time"
)

func main() {
	// Parse command-line flags
	timeout := flag.Duration("timeout", 10*time.Second, "connection timeout")
	flag.Parse()

	// Check for required arguments (host and port)
	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--timeout=<duration>] <host> <port>\n", os.Args[0])
		os.Exit(1)
	}
	host := flag.Args()[0]
	port := flag.Args()[1]
	addr := net.JoinHostPort(host, port)

	// Establish TCP connection with timeout
	conn, err := net.DialTimeout("tcp", addr, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("Connected to %s\n", addr)

	// Channel to signal connection closure
	done := make(chan struct{})

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, closing connection...")
		conn.Close()
		close(done)
	}()

	// Start goroutines for concurrent I/O
	go readFromConn(conn, done)
	go writeToConn(conn, done)

	// Wait for done signal
	<-done
}

func readFromConn(conn net.Conn, done chan struct{}) {
	reader := bufio.NewReader(conn)
	for {
		// Read until newline or EOF
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error reading from connection: %v\n", err)
			} else {
				fmt.Println("Connection closed by server")
			}
			conn.Close()
			close(done)
			return
		}
		fmt.Print(line)
	}
}

func writeToConn(conn net.Conn, done chan struct{}) {
	reader := bufio.NewReader(os.Stdin)
	for {
		// Read user input from STDIN
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nReceived EOF (Ctrl+D), closing connection...")
			} else {
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			}
			conn.Close()
			close(done)
			return
		}
		// Write to connection
		_, err = conn.Write([]byte(line))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to connection: %v\n", err)
			conn.Close()
			close(done)
			return
		}
	}
}
