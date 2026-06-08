package nvimbridge

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

func requireNvim(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed; skipping embedded integration test")
	}
}

// waitFor polls cond until it holds or the deadline passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestEmbeddedRoundTrip(t *testing.T) {
	requireNvim(t)

	b, err := Start("nvim", 40, 6)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer b.Close()

	const sql = "select 1;\nselect 2;"
	if err := b.SetText(sql); err != nil {
		t.Fatalf("SetText: %v", err)
	}

	// Mirror is updated synchronously by SetText.
	if got := b.Text(); got != sql {
		t.Errorf("Text() = %q, want %q", got, sql)
	}
	// Live read goes back to nvim and must agree.
	if got, err := b.LiveText(); err != nil || got != sql {
		t.Errorf("LiveText() = %q, %v; want %q", got, err, sql)
	}

	// Render must always produce exactly the requested number of rows.
	if rows := strings.Count(b.Render(40, 6, true), "\n") + 1; rows != 6 {
		t.Errorf("Render produced %d rows, want 6", rows)
	}
}

func TestEmbeddedInputEditsBuffer(t *testing.T) {
	requireNvim(t)

	b, err := Start("nvim", 40, 6)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer b.Close()

	if err := b.SetText("world"); err != nil {
		t.Fatalf("SetText: %v", err)
	}

	// Enter insert mode at the start of the line, prepend text, leave insert mode.
	b.Input("ggI" + "hello <Esc>")

	waitFor(t, "buffer to reflect typed text", func() bool {
		got, _ := b.LiveText()
		return got == "hello world"
	})

	// A flush should have arrived on the redraw channel.
	waitFor(t, "a redraw signal", func() bool {
		select {
		case <-b.RedrawCh():
			return true
		default:
			return false
		}
	})
}

func TestEmbeddedCursorTracking(t *testing.T) {
	requireNvim(t)

	b, err := Start("nvim", 40, 6)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer b.Close()

	if err := b.SetText("a\nb\nc\nd"); err != nil {
		t.Fatalf("SetText: %v", err)
	}

	b.Input("G") // jump to last line

	waitFor(t, "cursor to reach last line", func() bool {
		return b.LiveCursorLine() == 3
	})
}
