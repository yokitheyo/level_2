package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	timeout := flag.Duration("timeout", 10*time.Second, "connection timeout")
	flag.Parse()

	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--timeout=<duration>] <host> <port>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands: \\quit or \\exit to close connection, Ctrl+C to interrupt\n")
		os.Exit(1)
	}
	host := flag.Args()[0]
	port := flag.Args()[1]
	addr := net.JoinHostPort(host, port)

	conn, err := net.DialTimeout("tcp", addr, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("Connected to %s\n", addr)

	done := make(chan struct{})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, closing connection...")
		conn.Close()
		close(done)
	}()

	go readFromConn(conn, done)
	go writeToConn(conn, done)

	<-done
}

func readFromConn(conn net.Conn, done chan struct{}) {
	defer func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}()

	reader := bufio.NewReader(conn)
	for {
		select {
		case <-done:
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Error reading from connection: %v\n", err)
			} else {
				fmt.Println("Connection closed by server")
			}
			return
		}
		fmt.Print(line)
	}
}

func writeToConn(conn net.Conn, done chan struct{}) {
	defer func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-done:
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nReceived EOF (Ctrl+D), closing connection...")
			} else {
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			}
			return
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "\\quit" || trimmed == "\\exit" {
			fmt.Println("Closing connection...")
			return
		}

		_, err = conn.Write([]byte(line))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to connection: %v\n", err)
			return
		}
	}
}
