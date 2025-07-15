package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func handleBuiltin(args []string) bool {
	switch args[0] {
	case "cd":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "cd: missing argument")
			return true
		}
		if err := os.Chdir(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		}
		return true
	case "pwd":
		if dir, err := os.Getwd(); err != nil {
			fmt.Fprintf(os.Stderr, "pwd: %v\n", err)
		} else {
			fmt.Println(dir)
		}
		return true
	case "echo":
		fmt.Println(strings.Join(args[1:], " "))
		return true
	case "kill":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "kill: need PID")
			return true
		}
		pid, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "kill: invalid PID")
			return true
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: not found PID %d: %v\n", pid, err)
			return true
		}

		err = proc.Kill()
		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: error kill proccess : %v\n", err)
		}
		return true
	case "ps":
		cmd := exec.Command("ps")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "ps: %v\n", err)
		}
		return true
	}
	return false
}

func executeCommand(args []string) error {
	if handleBuiltin(args) {
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func killProcess(pid int) error {
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F").Run()
	}
	return syscall.Kill(pid, syscall.SIGKILL)
}

func handlePipeline(pipeline string) error {
	commands := strings.Split(pipeline, "|")
	var cmds []*exec.Cmd

	for _, cmdStr := range commands {
		args := strings.Fields(strings.TrimSpace(cmdStr))
		if len(args) == 0 {
			return fmt.Errorf("empty command in pipeline")
		}
		if handleBuiltin(args) {
			return nil
		}
		cmds = append(cmds, exec.Command(args[0], args[1:]...))
	}

	for i := 0; i < len(cmds)-1; i++ {
		r, w := io.Pipe()
		cmds[i].Stdout = w
		cmds[i+1].Stdin = r
		defer r.Close()
	}

	cmds[len(cmds)-1].Stdout = os.Stdout
	for _, cmd := range cmds {
		cmd.Stderr = os.Stderr
	}

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start command: %v", err)
		}
	}

	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("command failed: %v", err)
		}
	}

	return nil
}

func handleLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	if strings.Contains(line, "|") {
		if err := handlePipeline(line); err != nil {
			fmt.Fprintf(os.Stderr, "pipeline error: %v\n", err)
		}
	} else {
		args := strings.Fields(line)
		if err := executeCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "command error: %v\n", err)
		}
	}
}

func setupSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	go func() {
		for {
			<-sigChan
			fmt.Println("\nReceived SIGINT (use Ctrl+D to exit)")
		}
	}()
}

func main() {
	setupSignals()
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("minishell> ")

		line, err := reader.ReadString('\n')
		if err == io.EOF {
			fmt.Println("\nExiting...")
			break
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			continue
		}

		handleLine(line)
	}
}
