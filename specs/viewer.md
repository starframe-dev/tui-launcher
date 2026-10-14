# Viewer — Terminal Multiplexer

## Overview
Terminal multiplexer with tab support, similar to Zellij/screen/tmux.

## Features

### Tab Bar
- [x] Tab list at top
- [x] Active tab highlighted
- [x] Close button [x] per tab
- [x] New tab button [+]
- [x] Tab name = command or "Shell"

### Terminal Panes
- [x] Interactive terminal with cursor support
- [x] PTY for full terminal emulation
- [x] ANSI escape codes support

### Navigation
- [x] Ctrl+Tab — next tab
- [x] Ctrl+Shift+Tab — prev tab
- [x] Alt+1-9 — switch to tab by number
- [x] Ctrl+W — close active tab
- [x] Ctrl+N — new tab

### Layout
```
┌─────────────────────────────────────────┐
│ [+] Shell [x] | Task 1 [x] | Task 2 [x] │
├─────────────────────────────────────────┤
│                                         │
│  Interactive terminal content            │
│  $ command                              │
│  cursor here█                           │
│                                         │
└─────────────────────────────────────────┘
```

## Tech
- Go with bubbletea (tcell backend)
- github.com/creack/pty for PTY
- github.com/mattn/go-runewidth for unicode width

## Data Model

```go
type Tab struct {
    ID      int
    Name    string
    Command string
    Dir     string
    PTY     *pty.PTY
    Width   int
    Height  int
}

type Model struct {
    Tabs    []Tab
    Active  int
    Width   int
    Height  int
}
```

## Keyboard Shortcuts

| Shortcut       | Action           |
|----------------|------------------|
| Ctrl+Tab       | Next tab         |
| Ctrl+Shift+Tab | Prev tab         |
| Alt+1-9        | Go to tab N      |
| Ctrl+W         | Close tab        |
| Ctrl+N         | New tab (shell)  |
| Ctrl+C         | Send Ctrl+C      |
| Ctrl+D         | Send Ctrl+D      |

## Running
```bash
go run ./viewer
```
