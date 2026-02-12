package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "minishell",
		Short: "minishell",
		RunE: func(cmd *cobra.Command, args []string) error {
			run()
			return nil
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Print(cwd + "> ")

		if !scanner.Scan() {
			// EOF (Ctrl+D)
			fmt.Println()
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		line = expandEnvVars(line)

		ctx, cancel := context.WithCancel(context.Background())
		sigCh := make(chan os.Signal, 1)
		if runtime.GOOS == "windows" {
			signal.Notify(sigCh, os.Interrupt)
		} else {
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		}

		go func() {
			s := <-sigCh
			if s == syscall.SIGQUIT {
				os.Exit(0)
			}
			cancel()
		}()

		_ = runCommand(ctx, line)
		// fmt.Printf("exit code: %d\n", code)

		signal.Stop(sigCh)
		close(sigCh)
		cancel()
	}
}

func expandEnvVars(line string) string {
	return os.ExpandEnv(line)
}

func runCommand(ctx context.Context, input string) int {
	parts := splitByConditional(input)
	code := 0
	for i, part := range parts {
		cond := part.cond
		cmd := part.cmd
		if i > 0 {
			if cond == "&&" && code != 0 {
				break
			}
			if cond == "||" && code == 0 {
				break
			}
		}
		code = runSingleCommand(ctx, cmd)
	}
	return code
}

type condPart struct {
	cmd  string
	cond string // "", "&&", "||"
}

func splitByConditional(input string) []condPart {
	var parts []condPart
	var cur strings.Builder
	cond := ""
	for i := 0; i < len(input); {
		if i+1 < len(input) && input[i] == '&' && input[i+1] == '&' {
			parts = append(parts, condPart{cmd: strings.TrimSpace(cur.String()), cond: cond})
			cur.Reset()
			cond = "&&"
			i += 2
			continue
		}
		if i+1 < len(input) && input[i] == '|' && input[i+1] == '|' {
			parts = append(parts, condPart{cmd: strings.TrimSpace(cur.String()), cond: cond})
			cur.Reset()
			cond = "||"
			i += 2
			continue
		}
		cur.WriteByte(input[i])
		i++
	}
	parts = append(parts, condPart{cmd: strings.TrimSpace(cur.String()), cond: cond})
	return parts
}

func runSingleCommand(ctx context.Context, input string) int {
	if strings.Contains(input, "|") {
		parts := splitPipe(input)
		return runPipelineWithRedirects(ctx, parts)
	}
	return runSimpleWithRedirects(ctx, input)
}

func runSimpleWithRedirects(ctx context.Context, input string) int {
	args, inFile, outFile := parseRedirects(input)
	if len(args) == 0 {
		return 0
	}

	switch args[0] {
	case "cd":
		return cmdCd(args)
	case "pwd":
		return cmdPwd(os.Stdout)
	case "echo":
		return cmdEcho(args, os.Stdout)
	case "kill":
		return cmdKill(ctx, args)
	case "ps":
		return cmdPs(ctx, args, os.Stdout)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// stdin
	if inFile != "" {
		f, err := os.Open(inFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "open:", err)
			return 1
		}
		defer f.Close()
		cmd.Stdin = f
	} else {
		cmd.Stdin = os.Stdin
	}

	// stdout
	if outFile != "" {
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "create:", err)
			return 1
		}
		defer f.Close()
		cmd.Stdout = f
	} else {
		cmd.Stdout = os.Stdout
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

func parseRedirects(input string) (args []string, inFile, outFile string) {
	tokens := splitArgs(input)

	var result []string

	for i := 0; i < len(tokens); i++ {
		if tokens[i] == ">" && i+1 < len(tokens) {
			outFile = tokens[i+1]
			i++
			continue
		}

		if tokens[i] == "<" && i+1 < len(tokens) {
			inFile = tokens[i+1]
			i++
			continue
		}

		result = append(result, tokens[i])
	}

	return result, inFile, outFile
}

func runPipelineWithRedirects(ctx context.Context, parts []string) int {
	n := len(parts)
	if n == 0 {
		return 0
	}

	var cmds []*exec.Cmd
	var inFile, outFile string

	for i, p := range parts {
		args, in, out := parseRedirects(p)
		if i == 0 && in != "" {
			inFile = in
		}
		if i == n-1 && out != "" {
			outFile = out
		}
		cmds = append(cmds, exec.CommandContext(ctx, args[0], args[1:]...))
	}

	// stdin
	if inFile != "" {
		f, err := os.Open(inFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "open:", err)
			return 1
		}
		defer f.Close()
		cmds[0].Stdin = f
	} else {
		cmds[0].Stdin = os.Stdin
	}

	// stdout
	if outFile != "" {
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "create:", err)
			return 1
		}
		defer f.Close()

		cmds[n-1].Stdout = f
	} else {
		cmds[n-1].Stdout = os.Stdout
	}

	cmds[n-1].Stderr = os.Stderr

	for i := 0; i < n-1; i++ {
		r, w := io.Pipe()
		cmds[i].Stdout = w
		cmds[i].Stderr = os.Stderr
		cmds[i+1].Stdin = r
	}

	for _, c := range cmds {
		if err := c.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "start:", err)
			return 1
		}
	}

	exit := 0
	for _, c := range cmds {
		if err := c.Wait(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				exit = ee.ExitCode()
			} else {
				fmt.Fprintln(os.Stderr, "wait:", err)
				return 1
			}
		}
	}

	return exit
}

func splitPipe(s string) []string {
	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return parts
}

func splitArgs(s string) []string {
	var args []string
	var cur strings.Builder
	inQuote := rune(0)
	esc := false

	for _, r := range s {
		if esc {
			cur.WriteRune(r)
			esc = false
			continue
		}

		if r == '\\' {
			esc = true
			continue
		}

		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			} else {
				cur.WriteRune(r)
			}
			continue
		}

		if r == '\'' || r == '"' {
			inQuote = r
			continue
		}

		if r == ' ' || r == '\t' {
			if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}

	if cur.Len() > 0 {
		args = append(args, cur.String())
	}

	return args
}

func cmdCd(args []string) int {
	if len(args) < 2 {
		fmt.Println("cd: missing operand")
		return 1
	}

	path := args[1]
	if !filepath.IsAbs(path) {
		cwd, _ := os.Getwd()
		path = filepath.Join(cwd, path)
	}

	if err := os.Chdir(path); err != nil {
		fmt.Println("cd:", err)
		return 1
	}

	return 0
}

func cmdPwd(w io.Writer) int {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(w, "pwd:", err)
		return 1
	}

	fmt.Fprintln(w, dir)

	return 0
}

func cmdEcho(args []string, w io.Writer) int {
	if len(args) <= 1 {
		fmt.Fprintln(w)
		return 0
	}

	fmt.Fprintln(w, strings.Join(args[1:], " "))

	return 0
}

func cmdKill(ctx context.Context, args []string) int {
	if len(args) < 2 {
		fmt.Println("kill: missing pid")
		return 1
	}

	pid := args[1]

	if runtime.GOOS == "windows" {
		cmd := exec.CommandContext(ctx, "taskkill", "/PID", pid, "/F")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			fmt.Fprintln(os.Stderr, err)

			return 1
		}

		return 0
	}

	cmd := exec.CommandContext(ctx, "kill", pid)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	return 0
}

func cmdPs(ctx context.Context, args []string, w io.Writer) int {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "tasklist")
	} else {
		if len(args) > 1 {
			cmd = exec.CommandContext(ctx, "ps", args[1:]...)
		} else {
			cmd = exec.CommandContext(ctx, "ps", "aux")
		}
	}

	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	return 0
}
