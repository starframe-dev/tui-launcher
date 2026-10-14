package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// Parse flags and session name from all args
	args := os.Args[1:]
	sessionName := "default"
	var cmdArgs []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session", "-s":
			if i+1 < len(args) {
				sessionName = args[i+1]
				i++
			}
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		default:
			cmdArgs = append(cmdArgs, args[i])
		}
	}

	// Set session for socket path
	setSessionName(sessionName)

	if len(cmdArgs) == 0 {
		// TUI mode
		runTUI()
		return
	}

	// CLI mode
	runCLI(cmdArgs)
}

func printHelp() {
	fmt.Print(`Viewer - Terminal Multiplexer

Usage:
  viewer [-s name]                    Start TUI with session name (default: "default")
  viewer [-s name] new-tab <name> [cmd]  Create new tab with name
  viewer [-s name] restart-tab <name> <cmd>  Restart tab by name (or create new)
  viewer [-s name] close-tab [idx]    Close tab
  viewer [-s name] switch-tab <idx>   Switch to tab
  viewer [-s name] list-tabs          List tabs
  viewer [-s name] send-keys <keys>   Send keys to active tab
  viewer [-s name] send-cmd <cmd>     Send command to active tab (adds \r)

Options:
  -s, --session <name>  Session name (default: "default")
  -h, --help            Show help

Examples:
  viewer -s myproject
  viewer -s myproject new-tab build "npm run build"
  viewer -s myproject restart-tab build "npm run build"
  viewer -s myproject send-cmd "git status"
`)
}

func runCLI(args []string) {
	if len(args) == 0 {
		printHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "new-tab":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: viewer new-tab <name> [cmd]")
			os.Exit(1)
		}
		name := args[1]
		cmd := "bash"
		if len(args) > 2 {
			cmd = strings.Join(args[2:], " ")
		}
		cliNewTab(name, cmd)
	case "restart-tab":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: viewer restart-tab <name> <cmd>")
			os.Exit(1)
		}
		name := args[1]
		cmd := strings.Join(args[2:], " ")
		cliRestartTab(name, cmd)
	case "close-tab":
		idx := -1
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &idx)
		}
		cliCloseTab(idx)
	case "switch-tab":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: viewer switch-tab <index>")
			os.Exit(1)
		}
		idx := 0
		fmt.Sscanf(args[1], "%d", &idx)
		cliSwitchTab(idx)
	case "list-tabs":
		cliListTabs()
	case "send-keys":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: viewer send-keys <keys>")
			os.Exit(1)
		}
		keys := strings.Join(args[1:], " ")
		cliSendKeys(keys)
	case "send-cmd":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: viewer send-cmd <command>")
			os.Exit(1)
		}
		cmd := strings.Join(args[1:], " ") + "\r"
		cliSendKeys(cmd)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		printHelp()
		os.Exit(1)
	}
}
