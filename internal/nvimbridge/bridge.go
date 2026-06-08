// Package nvimbridge embeds a real Neovim process and exposes it as a panel
// inside the TUI. It speaks the msgpack-rpc remote-UI protocol over the child
// process's stdio (nvim --embed): redraw notifications drive a cell grid that
// Render turns into a styled string, and key presses are forwarded verbatim via
// nvim_input. The embedded editor is a scratch SQL buffer, so the host app can
// read its contents (Text/LiveText) to execute statements and run completions
// while the user gets their real Vim — motions, plugins, and init.lua included.
package nvimbridge

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/neovim/go-client/nvim"
)

// Bridge owns the embedded Neovim process and the mirrored UI state.
type Bridge struct {
	v   *nvim.Nvim
	buf nvim.Buffer // dedicated scratch SQL buffer

	// Grid state, written by the redraw notification goroutine and read by the
	// renderer. Guarded by gridMu.
	gridMu              sync.Mutex
	cols, rows          int
	cells               [][]cell
	curRow, curCol      int
	defFg, defBg, defSp int
	hls                 map[int]hlAttr
	mode                string

	// Buffer mirror, kept current via autocmd notifications. Guarded by linesMu.
	linesMu    sync.Mutex
	lines      []string
	cursorLine int // 0-based
	cursorCol  int // 0-based byte column

	redraw    chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	deadErr   error
}

// Start launches `binary --embed`, attaches as a remote UI of the given size,
// turns the buffer into a scratch SQL buffer, and wires up the change/cursor
// notifications. It returns an error (and leaves nothing running) if the binary
// is missing or the handshake fails, so callers can fall back gracefully.
func Start(binary string, width, height int) (*Bridge, error) {
	if binary == "" {
		binary = "nvim"
	}
	if _, err := exec.LookPath(binary); err != nil {
		return nil, fmt.Errorf("nvimbridge: %q not found: %w", binary, err)
	}
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}

	v, err := nvim.NewChildProcess(
		nvim.ChildProcessCommand(binary),
		nvim.ChildProcessArgs("--embed", "-n"),
		nvim.ChildProcessServe(false),
	)
	if err != nil {
		return nil, fmt.Errorf("nvimbridge: spawn: %w", err)
	}

	b := &Bridge{
		v:      v,
		defFg:  -1,
		defBg:  -1,
		defSp:  -1,
		hls:    map[int]hlAttr{},
		mode:   "normal",
		redraw: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
	b.allocGrid(width, height)

	if err := v.RegisterHandler("redraw", b.handleRedraw); err != nil {
		_ = v.Close()
		return nil, fmt.Errorf("nvimbridge: register redraw: %w", err)
	}
	_ = v.RegisterHandler("seshat_cursor", b.handleCursor)
	_ = v.RegisterHandler("seshat_lines", b.handleLines)

	// Serve must run before any request (including AttachUI) can complete, since
	// responses are delivered by the serve loop. Handlers are registered first so
	// the initial redraw burst is never dropped.
	go func() { b.markDead(v.Serve()) }()

	if err := v.AttachUI(width, height, map[string]any{
		"rgb":          true,
		"ext_linegrid": true,
	}); err != nil {
		_ = v.Close()
		b.markDead(err)
		return nil, fmt.Errorf("nvimbridge: ui_attach: %w", err)
	}

	if err := b.setup(); err != nil {
		_ = v.Close()
		b.markDead(err)
		return nil, fmt.Errorf("nvimbridge: setup: %w", err)
	}
	return b, nil
}

// setup creates a dedicated scratch SQL buffer (always modifiable, regardless
// of any start-screen plugin in the user's config), switches to it, and
// installs buffer-local autocmds that push buffer text and cursor position back
// over RPC, keeping the Go-side mirror current without polling.
func (b *Bridge) setup() error {
	buf, err := b.v.CreateBuffer(false, true) // unlisted, scratch
	if err != nil {
		return err
	}
	if err := b.v.SetCurrentBuffer(buf); err != nil {
		return err
	}
	b.buf = buf

	lua := fmt.Sprintf(`
local chan = %d
local buf = %d
vim.bo[buf].filetype = 'sql'
local grp = vim.api.nvim_create_augroup('seshat_bridge', { clear = true })
vim.api.nvim_create_autocmd({ 'CursorMoved', 'CursorMovedI' }, {
  group = grp,
  buffer = buf,
  callback = function()
    local ok, pos = pcall(vim.api.nvim_win_get_cursor, 0)
    if ok then vim.rpcnotify(chan, 'seshat_cursor', pos[1] - 1, pos[2]) end
  end,
})
vim.api.nvim_create_autocmd({ 'TextChanged', 'TextChangedI', 'InsertEnter', 'InsertLeave' }, {
  group = grp,
  buffer = buf,
  callback = function()
    vim.rpcnotify(chan, 'seshat_lines', vim.api.nvim_buf_get_lines(buf, 0, -1, false))
  end,
})
`, b.v.ChannelID(), int(buf))
	return b.v.ExecLua(lua, nil)
}

func (b *Bridge) handleCursor(args ...any) {
	if len(args) == 0 {
		return
	}
	b.linesMu.Lock()
	b.cursorLine = asInt(args[0])
	if len(args) > 1 {
		b.cursorCol = asInt(args[1])
	}
	b.linesMu.Unlock()
}

func (b *Bridge) handleLines(args ...any) {
	if len(args) == 0 {
		return
	}
	arr := asSlice(args[0])
	lines := make([]string, len(arr))
	for i, l := range arr {
		lines[i] = asString(l)
	}
	b.linesMu.Lock()
	b.lines = lines
	b.linesMu.Unlock()
}

func (b *Bridge) markDead(err error) {
	b.closeOnce.Do(func() {
		b.deadErr = err
		close(b.done)
	})
}

// RedrawCh receives a value after every flushed frame (coalesced; never closed).
func (b *Bridge) RedrawCh() <-chan struct{} { return b.redraw }

// Done is closed when the Neovim process exits or the bridge is closed.
func (b *Bridge) Done() <-chan struct{} { return b.done }

// Close terminates the embedded Neovim process.
func (b *Bridge) Close() error {
	err := b.v.Close()
	b.markDead(err)
	return err
}

// Input forwards a key sequence in Neovim notation (e.g. "<Esc>", "<C-w>", "i").
func (b *Bridge) Input(keys string) {
	if keys == "" {
		return
	}
	_, _ = b.v.Input(keys)
}

// Resize tells Neovim the UI grid is now w x cols by h rows.
func (b *Bridge) Resize(w, h int) {
	if w < 1 || h < 1 {
		return
	}
	_ = b.v.TryResizeUI(w, h)
}

// ModeName returns the last reported editor mode ("normal", "insert", ...).
func (b *Bridge) ModeName() string {
	b.gridMu.Lock()
	defer b.gridMu.Unlock()
	return b.mode
}

// Text returns the mirrored buffer contents (fast; safe to call every frame).
func (b *Bridge) Text() string {
	b.linesMu.Lock()
	defer b.linesMu.Unlock()
	return strings.Join(b.lines, "\n")
}

// CursorLine returns the mirrored 0-based cursor line (fast).
func (b *Bridge) CursorLine() int {
	b.linesMu.Lock()
	defer b.linesMu.Unlock()
	return b.cursorLine
}

// CursorCol returns the mirrored 0-based byte column of the cursor (fast).
func (b *Bridge) CursorCol() int {
	b.linesMu.Lock()
	defer b.linesMu.Unlock()
	return b.cursorCol
}

// CursorScreen returns the cursor position within the rendered grid (row, col).
func (b *Bridge) CursorScreen() (row, col int) {
	b.gridMu.Lock()
	defer b.gridMu.Unlock()
	return b.curRow, b.curCol
}

// LiveText reads the buffer straight from Neovim (blocking RPC) for operations
// that must see the exact current contents, e.g. executing a statement. It also
// refreshes the mirror. Falls back to the mirror on error.
func (b *Bridge) LiveText() (string, error) {
	lines, err := b.v.BufferLines(b.buf, 0, -1, false)
	if err != nil {
		return b.Text(), err
	}
	parts := make([]string, len(lines))
	for i, l := range lines {
		parts[i] = string(l)
	}
	b.linesMu.Lock()
	b.lines = parts
	b.linesMu.Unlock()
	return strings.Join(parts, "\n"), nil
}

// LiveCursorLine reads the 0-based cursor line straight from Neovim.
func (b *Bridge) LiveCursorLine() int {
	win, err := b.v.CurrentWindow()
	if err != nil {
		return b.CursorLine()
	}
	pos, err := b.v.WindowCursor(win)
	if err != nil {
		return b.CursorLine()
	}
	return pos[0] - 1
}

// SetText replaces the entire buffer and resets the mirror.
func (b *Bridge) SetText(s string) error {
	parts := strings.Split(s, "\n")
	repl := make([][]byte, len(parts))
	for i, p := range parts {
		repl[i] = []byte(p)
	}
	if err := b.v.SetBufferLines(b.buf, 0, -1, false, repl); err != nil {
		return err
	}
	b.linesMu.Lock()
	b.lines = parts
	b.cursorLine = 0
	b.cursorCol = 0
	b.linesMu.Unlock()
	return nil
}
