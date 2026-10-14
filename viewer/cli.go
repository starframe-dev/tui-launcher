package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func sendCommand(cmd string) error {
	socketPath := getSocketPath()
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("connect to viewer: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write([]byte(cmd + "\n"))
	return err
}

func cliNewTab(name, cmd string) {
	if err := sendCommand("NEW_TAB " + name + " " + cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Tab created")
}

func cliRestartTab(name, cmd string) {
	if err := sendCommand("RESTART_TAB " + name + " " + cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Tab restarted")
}

func cliCloseTab(idx int) {
	cmd := "CLOSE_TAB"
	if idx >= 0 {
		cmd += fmt.Sprintf(" %d", idx)
	}
	if err := sendCommand(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Tab closed")
}

func cliSwitchTab(idx int) {
	if err := sendCommand(fmt.Sprintf("SWITCH_TAB %d", idx)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Tab switched")
}

func cliListTabs() {
	fmt.Println("Active tabs:")
	fmt.Println("(Run 'viewer' in TUI mode to see tabs)")
}

func cliSendKeys(keys string) {
	if err := sendCommand("SEND_KEYS " + keys); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Keys sent")
}
