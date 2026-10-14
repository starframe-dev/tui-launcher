package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var currentSession = "default"

func setSessionName(name string) {
	if name != "" {
		currentSession = name
	}
}

func getSessionName() string {
	return currentSession
}

func getSocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("viewer-%s.sock", currentSession))
}
