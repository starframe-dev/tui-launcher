package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type Server struct {
	app        *App
	listener   net.Listener
	socketPath string
}

func NewServer(app *App) *Server {
	return &Server{
		app:        app,
		socketPath: getSocketPath(),
	}
}

func (s *Server) Start() error {
	os.Remove(s.socketPath)

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = listener
	os.Chmod(s.socketPath, 0666)

	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		// Only trim trailing newline, preserve \r (which is Enter for PTY)
		line = strings.TrimSuffix(line, "\n")
		if line == "" {
			continue
		}
		s.handleCommand(line)
	}
}

// Command format: CMD name command args...
func (s *Server) handleCommand(cmd string) {
	parts := strings.SplitN(cmd, " ", 2)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "NEW_TAB":
		name, command := parseNamedCommand(parts)
		s.app.NewTab(name, command)
	case "RESTART_TAB":
		name, command := parseNamedCommand(parts)
		if tab := s.app.FindTabByName(name); tab != nil {
			tab.Restart(command)
		} else {
			s.app.NewTab(name, command)
		}
	case "CLOSE_TAB":
		idx := s.app.activeIdx
		if len(parts) > 1 {
			if i, err := parseInt(parts[1]); err == nil {
				idx = i
			}
		}
		s.app.CloseTab(idx)
	case "SWITCH_TAB":
		if len(parts) > 1 {
			if i, err := parseInt(parts[1]); err == nil {
				s.app.SwitchTab(i)
			}
		}
	case "SEND_KEYS":
		if len(parts) > 1 {
			s.app.SendKeys(parts[1])
		}
	}
}

// parseNamedCommand parses "name command args..." from command parts
func parseNamedCommand(parts []string) (string, string) {
	if len(parts) < 2 {
		return "", ""
	}
	// Format: "name cd /dir && command"
	rest := parts[1]
	nameEnd := strings.IndexByte(rest, ' ')
	if nameEnd < 0 {
		return rest, ""
	}
	return rest[:nameEnd], rest[nameEnd+1:]
}

func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
	os.Remove(s.socketPath)
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
