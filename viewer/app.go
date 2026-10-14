package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

var (
	tabBarBg      = lipgloss.Color("#1a1a2e")
	activeTabBg   = lipgloss.Color("#16213e")
	activeTabFg   = lipgloss.Color("#00d4ff")
	inactiveTabFg = lipgloss.Color("#888888")
	newTabFg      = lipgloss.Color("#50fa7b")
)

const maxHistoryLines = 10000

type ptyMsg struct {
	tabID int
}

type ptyClosedMsg struct {
	tabID int
}

type tabRegion struct {
	idx    int
	startX int
	endX   int
	closeX int
}

type newTabRegion struct {
	startX int
	endX   int
}

type point struct {
	x int
	y int
}

type Tab struct {
	id      int
	name    string
	command string
	pty     *os.File
	cmd     *exec.Cmd
	vt      *vt.Emulator
	width   int
	height  int
	app     *App
	mu      sync.Mutex

	scrollOffset int
	restarting   bool
}

type App struct {
	tabs         []*Tab
	activeIdx    int
	width        int
	height       int
	tabID        int
	server       *Server
	program      *tea.Program
	tabRegions   []tabRegion
	newTabRegion *newTabRegion

	selecting      bool
	selectionStart point
	selectionEnd   point
	selectionText  string
	wheelRemainder float64
}

func NewApp() *App {
	return &App{tabs: make([]*Tab, 0)}
}

func (a *App) Init() tea.Cmd {
	a.NewTab("", "")
	go a.server.Start()
	return nil
}

func newTerminal(w, h int) *vt.Emulator {
	emu := vt.NewEmulator(w, h)
	emu.SetScrollbackSize(maxHistoryLines)
	return emu
}

func startShell(tab *Tab, command string) {
	var cmd *exec.Cmd
	if command == "bash" || command == "zsh" || command == "sh" {
		cmd = exec.Command(command, "-i")
	} else {
		cmd = exec.Command("bash", "-i", "-c", command)
	}
	cmd.Env = os.Environ()

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		tab.writeVT([]byte(fmt.Sprintf("\r\nError: %v\r\n", err)))
		return
	}

	tab.pty = ptyFile
	tab.cmd = cmd
	go func() {
		cmd.Wait()
		if tab.pty != nil {
			tab.pty.Close()
			tab.pty = nil
		}
	}()
	go tab.readLoop()
}

func (t *Tab) writeVT(data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, _ = t.vt.Write(data)
}

func (a *App) NewTab(name, command string) {
	a.tabID++
	if command == "" {
		command = os.Getenv("SHELL")
		if command == "" {
			command = "bash"
		}
	}
	if name == "" {
		name = getCommandName(command)
	}

	w, h := 80, 24
	if a.width > 0 {
		w = a.width
	}
	if a.height > 0 {
		h = a.height - 1
	}

	tab := &Tab{
		id:      a.tabID,
		name:    name,
		command: command,
		vt:      newTerminal(w, h),
		width:   w,
		height:  h,
		app:     a,
	}

	startShell(tab, command)

	insertIdx := len(a.tabs)
	for i, t := range a.tabs {
		if t.name == name {
			insertIdx = i + 1
			break
		}
	}

	if insertIdx >= len(a.tabs) {
		a.tabs = append(a.tabs, tab)
	} else {
		a.tabs = append(a.tabs[:insertIdx+1], a.tabs[insertIdx:]...)
		a.tabs[insertIdx] = tab
	}
	a.activeIdx = insertIdx
}

func (a *App) NewTabAfter(name, command, afterName string) {
	a.tabID++
	if command == "" {
		command = os.Getenv("SHELL")
		if command == "" {
			command = "bash"
		}
	}
	if name == "" {
		name = getCommandName(command)
	}

	w, h := 80, 24
	if a.width > 0 {
		w = a.width
	}
	if a.height > 0 {
		h = a.height - 1
	}

	tab := &Tab{
		id:      a.tabID,
		name:    name,
		command: command,
		vt:      newTerminal(w, h),
		width:   w,
		height:  h,
		app:     a,
	}

	startShell(tab, command)

	insertIdx := len(a.tabs)
	for i, t := range a.tabs {
		if t.name == afterName {
			insertIdx = i + 1
			break
		}
	}

	if insertIdx >= len(a.tabs) {
		a.tabs = append(a.tabs, tab)
	} else {
		a.tabs = append(a.tabs[:insertIdx+1], a.tabs[insertIdx:]...)
		a.tabs[insertIdx] = tab
	}
	a.activeIdx = insertIdx
}

func (t *Tab) readLoop() {
	buf := make([]byte, 4096)
	for {
		if t.pty == nil {
			return
		}
		n, err := t.pty.Read(buf)
		if err != nil {
			if !t.restarting && t.app != nil && t.app.program != nil {
				t.app.program.Send(ptyClosedMsg{tabID: t.id})
			}
			return
		}
		if n > 0 {
			t.writeVT(buf[:n])
			if t.app != nil && t.app.program != nil {
				t.app.program.Send(ptyMsg{tabID: t.id})
			}
		}
	}
}

func (t *Tab) Restart(command string) {
	t.restarting = true
	killProcessGroup(t.cmd)
	if t.pty != nil {
		t.pty.Close()
		t.pty = nil
	}
	time.Sleep(50 * time.Millisecond)

	t.mu.Lock()
	t.vt = newTerminal(t.width, t.height)
	t.mu.Unlock()

	t.command = command
	startShell(t, command)
	t.restarting = false
}

func (a *App) FindTabByName(name string) *Tab {
	for _, tab := range a.tabs {
		if tab.name == name {
			return tab
		}
	}
	return nil
}

func (a *App) CloseTab(idx int) {
	if idx < 0 || idx >= len(a.tabs) {
		return
	}
	tab := a.tabs[idx]
	killProcessGroup(tab.cmd)
	if tab.pty != nil {
		tab.pty.Close()
		tab.pty = nil
	}

	a.tabs = append(a.tabs[:idx], a.tabs[idx+1:]...)
	if a.activeIdx >= len(a.tabs) {
		a.activeIdx = len(a.tabs) - 1
	} else if idx < a.activeIdx {
		a.activeIdx--
	}
}

func (a *App) SwitchTab(idx int) {
	if idx >= 0 && idx < len(a.tabs) {
		a.activeIdx = idx
	}
}

func (a *App) NextTab() {
	if len(a.tabs) > 1 {
		a.activeIdx = (a.activeIdx + 1) % len(a.tabs)
	}
}

func (a *App) PrevTab() {
	if len(a.tabs) > 1 {
		a.activeIdx = (a.activeIdx - 1 + len(a.tabs)) % len(a.tabs)
	}
}

func (a *App) ActiveTab() *Tab {
	if a.activeIdx < 0 || a.activeIdx >= len(a.tabs) {
		return nil
	}
	return a.tabs[a.activeIdx]
}

func (a *App) SendKeys(keys string) {
	tab := a.ActiveTab()
	if tab == nil || tab.pty == nil {
		return
	}
	_, _ = tab.pty.Write([]byte(keys))
}

func (a *App) SendByte(b byte) {
	tab := a.ActiveTab()
	if tab == nil || tab.pty == nil {
		return
	}
	_, _ = tab.pty.Write([]byte{b})
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.selecting && msg.String() == "esc" {
			a.clearSelection()
			return a, nil
		}
		if msg.String() == "ctrl+y" && a.selectionText != "" {
			a.copyText(a.selectionText)
			return a, nil
		}
		if tab := a.ActiveTab(); tab != nil && tab.scrollOffset > 0 {
			switch msg.Type {
			case tea.KeyPgUp, tea.KeyPgDown:
			default:
				tab.scrollOffset = 0
			}
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			a.SendByte(3)
			return a, nil
		case tea.KeyCtrlD:
			a.SendByte(4)
			if len(a.tabs) == 1 {
				return a, tea.Quit
			}
			a.CloseTab(a.activeIdx)
			return a, nil
		case tea.KeyCtrlW:
			a.CloseTab(a.activeIdx)
			return a, nil
		case tea.KeyCtrlT:
			a.NewTab("", "")
			return a, nil
		case tea.KeyCtrlI:
			if msg.String() == "ctrl+tab" {
				a.NextTab()
				return a, nil
			}
			a.SendByte('\t')
			return a, nil
		case tea.KeyRunes:
			a.SendKeys(string(msg.Runes))
			return a, nil
		case tea.KeySpace:
			a.SendByte(' ')
			return a, nil
		case tea.KeyEnter:
			a.SendKeys("\r")
			return a, nil
		case tea.KeyBackspace:
			a.SendByte(127)
			return a, nil
		case tea.KeyDelete:
			a.SendKeys("\x1b[3~")
			return a, nil
		case tea.KeyEscape:
			a.SendByte(27)
			return a, nil
		case tea.KeyUp:
			a.SendKeys("\x1b[A")
			return a, nil
		case tea.KeyDown:
			a.SendKeys("\x1b[B")
			return a, nil
		case tea.KeyLeft:
			a.SendKeys("\x1b[D")
			return a, nil
		case tea.KeyRight:
			a.SendKeys("\x1b[C")
			return a, nil
		case tea.KeyHome:
			a.SendKeys("\x1b[H")
			return a, nil
		case tea.KeyEnd:
			a.SendKeys("\x1b[F")
			return a, nil
		case tea.KeyPgUp:
			a.scrollActiveTab(1)
			return a, nil
		case tea.KeyPgDown:
			a.scrollActiveTab(-1)
			return a, nil
		case tea.KeyF1:
			a.SendKeys("\x1bOP")
			return a, nil
		case tea.KeyF2:
			a.SendKeys("\x1bOQ")
			return a, nil
		case tea.KeyF3:
			a.SendKeys("\x1bOR")
			return a, nil
		case tea.KeyF4:
			a.SendKeys("\x1bOS")
			return a, nil
		case tea.KeyF5:
			a.SendKeys("\x1b[15~")
			return a, nil
		case tea.KeyF6:
			a.SendKeys("\x1b[17~")
			return a, nil
		case tea.KeyF7:
			a.SendKeys("\x1b[18~")
			return a, nil
		case tea.KeyF8:
			a.SendKeys("\x1b[19~")
			return a, nil
		case tea.KeyF9:
			a.SendKeys("\x1b[20~")
			return a, nil
		case tea.KeyF10:
			a.SendKeys("\x1b[21~")
			return a, nil
		case tea.KeyF11:
			a.SendKeys("\x1b[23~")
			return a, nil
		case tea.KeyF12:
			a.SendKeys("\x1b[24~")
			return a, nil
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		termH := a.height - 1
		for _, tab := range a.tabs {
			tab.width = a.width
			tab.height = termH
			tab.mu.Lock()
			tab.vt.Resize(a.width, termH)
			tab.mu.Unlock()
			if tab.pty != nil {
				_ = pty.Setsize(tab.pty, &pty.Winsize{Rows: uint16(termH), Cols: uint16(a.width)})
			}
		}
		return a, nil

	case ptyMsg:
		return a, nil

	case ptyClosedMsg:
		return a, nil

	case tea.MouseMsg:
		return a.handleMouse(msg), nil
	}
	return a, nil
}

func (a *App) handleMouse(msg tea.MouseMsg) *App {
	if msg.Y == 0 && msg.Action == tea.MouseActionPress {
		idx, isClose, isNew := a.tabHitTest(msg.X)
		if isNew {
			a.NewTab("", "")
			return a
		}
		if isClose && idx >= 0 {
			a.CloseTab(idx)
			return a
		}
		if idx >= 0 {
			a.SwitchTab(idx)
			return a
		}
	}

	if msg.Y <= 0 {
		return a
	}

	tab := a.ActiveTab()
	if tab == nil {
		return a
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		a.scrollWheel(1.0)
		return a
	case tea.MouseButtonWheelDown:
		a.scrollWheel(-1.0)
		return a
	}

	p := point{x: clamp(msg.X, 0, max(0, tab.width-1)), y: clamp(msg.Y-1, 0, max(0, tab.height-1))}
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		a.selecting = true
		a.selectionStart = p
		a.selectionEnd = p
		a.selectionText = ""
		return a
	}
	if a.selecting && msg.Action == tea.MouseActionMotion {
		a.selectionEnd = p
		return a
	}
	if a.selecting && msg.Action == tea.MouseActionRelease {
		a.selectionEnd = p
		a.selectionText = a.selectedText(tab)
		a.selecting = false
		if a.selectionText != "" {
			a.copyText(a.selectionText)
		}
		return a
	}
	return a
}

func (a *App) scrollWheel(delta float64) {
	const linesPerWheelEvent = 1.0 / 3.0

	a.wheelRemainder += delta * linesPerWheelEvent
	for a.wheelRemainder >= 1.0 {
		a.scrollActiveTab(1)
		a.wheelRemainder -= 1.0
	}
	for a.wheelRemainder <= -1.0 {
		a.scrollActiveTab(-1)
		a.wheelRemainder += 1.0
	}
}

func (a *App) scrollActiveTab(delta int) {
	tab := a.ActiveTab()
	if tab == nil {
		return
	}
	tab.mu.Lock()
	maxOffset := tab.vt.ScrollbackLen()
	tab.mu.Unlock()
	tab.scrollOffset = clamp(tab.scrollOffset+delta, 0, maxOffset)
}

func (a *App) clearSelection() {
	a.selecting = false
	a.selectionText = ""
}

func (a *App) findTab(id int) *Tab {
	for _, tab := range a.tabs {
		if tab.id == id {
			return tab
		}
	}
	return nil
}

func (a *App) tabHitTest(x int) (idx int, isClose bool, isNew bool) {
	for _, r := range a.tabRegions {
		if x >= r.startX && x < r.endX {
			if x == r.closeX {
				return r.idx, true, false
			}
			return r.idx, false, false
		}
	}
	if a.newTabRegion != nil && x >= a.newTabRegion.startX && x < a.newTabRegion.endX {
		return -1, false, true
	}
	return -1, false, false
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	var b strings.Builder
	b.WriteString(a.renderTabBar())
	b.WriteString("\n")
	if tab := a.ActiveTab(); tab != nil {
		b.WriteString(a.renderTerminal(tab))
	}
	return b.String()
}

func (a *App) renderTabBar() string {
	a.tabRegions = nil
	var parts []string
	col := 0
	for i, tab := range a.tabs {
		name := tab.name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		rawLabel := fmt.Sprintf(" %s ×", name)
		if i == a.activeIdx {
			rawLabel = fmt.Sprintf("▎ %s ×", name)
		}
		w := utf8.RuneCountInString(rawLabel)
		a.tabRegions = append(a.tabRegions, tabRegion{idx: i, startX: col, endX: col + w, closeX: col + w - 1})
		col += w
		style := lipgloss.NewStyle().Background(tabBarBg).Foreground(inactiveTabFg)
		if i == a.activeIdx {
			style = lipgloss.NewStyle().Background(activeTabBg).Foreground(activeTabFg).Bold(true)
		}
		parts = append(parts, style.Render(rawLabel))
	}

	newLabel := " + "
	newW := utf8.RuneCountInString(newLabel)
	a.newTabRegion = &newTabRegion{startX: col, endX: col + newW}
	col += newW
	parts = append(parts, lipgloss.NewStyle().Foreground(newTabFg).Render(newLabel))

	bar := lipgloss.NewStyle().Background(tabBarBg).Render(strings.Join(parts, ""))
	if padding := a.width - col; padding > 0 {
		bar += lipgloss.NewStyle().Background(tabBarBg).Render(strings.Repeat(" ", padding))
	}
	return bar
}

func (a *App) renderTerminal(tab *Tab) string {
	tab.mu.Lock()
	defer tab.mu.Unlock()

	if tab.scrollOffset > 0 {
		return a.renderScrollback(tab)
	}

	cursor := tab.vt.CursorPosition()
	lines := make([]string, 0, tab.height)
	for y := 0; y < tab.height; y++ {
		var b strings.Builder
		for x := 0; x < tab.width; x++ {
			selected := a.isSelected(x, y)
			if x == cursor.X && y == cursor.Y {
				selected = true
			}
			b.WriteString(renderCell(tab.vt.CellAt(x, y), selected))
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func (a *App) renderScrollback(tab *Tab) string {
	lines := make([]string, 0, tab.height)
	for y := 0; y < tab.height; y++ {
		var b strings.Builder
		for x := 0; x < tab.width; x++ {
			b.WriteString(renderCell(a.visibleCellAt(tab, x, y), a.isSelected(x, y)))
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func (a *App) visibleCellAt(tab *Tab, x, y int) *uv.Cell {
	sbLen := tab.vt.ScrollbackLen()
	totalLines := sbLen + tab.height
	start := totalLines - tab.height - tab.scrollOffset
	if start < 0 {
		start = 0
	}
	virtualY := start + y
	if virtualY < sbLen {
		return tab.vt.ScrollbackCellAt(x, virtualY)
	}
	return tab.vt.CellAt(x, virtualY-sbLen)
}

func renderCell(cell *uv.Cell, selected bool) string {
	content := " "
	if cell != nil {
		content = cell.String()
		if content == "" {
			content = " "
		}
	}
	if cell == nil || cell.Style.IsZero() {
		if selected {
			return "\x1b[7m" + content + "\x1b[0m"
		}
		return content
	}
	if selected {
		return cell.Style.String() + "\x1b[7m" + content + "\x1b[0m"
	}
	return cell.Style.Styled(content)
}

func (a *App) selectedText(tab *Tab) string {
	tab.mu.Lock()
	defer tab.mu.Unlock()

	start, end := normalizedSelection(a.selectionStart, a.selectionEnd)
	var lines []string
	for y := start.y; y <= end.y; y++ {
		fromX := 0
		toX := tab.width - 1
		if y == start.y {
			fromX = start.x
		}
		if y == end.y {
			toX = end.x
		}
		var b strings.Builder
		for x := fromX; x <= toX; x++ {
			b.WriteString(a.cellText(tab, x, y))
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func (a *App) cellText(tab *Tab, x, y int) string {
	cell := tab.vt.CellAt(x, y)
	if tab.scrollOffset > 0 {
		cell = a.visibleCellAt(tab, x, y)
	}
	if cell == nil || cell.String() == "" {
		return " "
	}
	return cell.String()
}

func (a *App) isSelected(x, y int) bool {
	if !a.selecting && a.selectionText == "" {
		return false
	}
	start, end := normalizedSelection(a.selectionStart, a.selectionEnd)
	if y < start.y || y > end.y {
		return false
	}
	if start.y == end.y {
		return x >= start.x && x <= end.x
	}
	if y == start.y {
		return x >= start.x
	}
	if y == end.y {
		return x <= end.x
	}
	return true
}

func normalizedSelection(a, b point) (point, point) {
	if a.y > b.y || (a.y == b.y && a.x > b.x) {
		return b, a
	}
	return a, b
}

func (a *App) copyText(text string) {
	if text == "" {
		return
	}
	cmd := exec.Command("pbcopy")
	stdin, err := cmd.StdinPipe()
	if err == nil && cmd.Start() == nil {
		_, _ = stdin.Write([]byte(text))
		_ = stdin.Close()
		if cmd.Wait() == nil {
			return
		}
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	fmt.Fprintf(os.Stdout, "\x1b]52;c;%s\a", encoded)
}

// killProcessGroup sends SIGTERM to the entire session, waits briefly,
// then sends SIGKILL if the process is still alive. This ensures child
// processes spawned by the shell are also terminated.
//
// pty.Start() creates a new session (setsid) so the session ID equals
// the process PID. Signalling the negative session ID reaches all
// processes in that session.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	sid := cmd.Process.Pid // session ID == PID because pty.Start uses setsid
	_ = syscall.Kill(-sid, syscall.SIGTERM)

	// Give the session a brief moment to shut down gracefully.
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(200 * time.Millisecond):
	}

	_ = syscall.Kill(-sid, syscall.SIGKILL)
	<-done
}

func getCommandName(command string) string {
	if command == "" {
		return "Shell"
	}
	parts := strings.Fields(command)
	if len(parts) > 0 {
		base := parts[0]
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		return base
	}
	return "Shell"
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func runTUI() {
	app := NewApp()
	app.server = NewServer(app)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	app.program = p
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, tab := range app.tabs {
		killProcessGroup(tab.cmd)
		if tab.pty != nil {
			tab.pty.Close()
		}
	}
	if app.server != nil {
		app.server.Stop()
	}
}
