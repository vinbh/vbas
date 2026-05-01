// Package ui renders an inline dropdown selector below the user's prompt.
//
// The dropdown opens /dev/tty for raw input and ANSI rendering so the
// caller's stdin/stdout/stderr stay clean. The terminal cursor is saved
// on entry (CSI s) and restored on exit (CSI u, then CSI J to clear
// the dropdown rows). All other keys are ignored — only Up/Down navigate,
// Tab/Enter select, Esc/Ctrl-C cancel.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

// Item is a single suggestion shown in the dropdown.
type Item struct {
	Value       string
	Description string
}

// MaxRows caps how many rows are visible at once. Items beyond this
// scroll into view as the user navigates with arrow keys.
const MaxRows = 10

// Run displays the dropdown for items and returns the selected
// item's Value, or "" if the user cancels (Esc / Ctrl-C / EOF).
//
// It opens /dev/tty itself; never write to os.Stdout from inside Run.
func Run(items []Item) (string, error) {
	if len(items) == 0 {
		return "", nil
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("open /dev/tty: %w", err)
	}
	defer tty.Close()

	old, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return "", fmt.Errorf("raw mode: %w", err)
	}
	defer term.Restore(int(tty.Fd()), old)

	d := &dropdown{
		items: items,
		tty:   tty,
		rows:  min(len(items), MaxRows),
	}

	d.init()
	defer d.cleanup()

	d.draw()
	for {
		k, err := d.readKey()
		if err != nil {
			return "", nil
		}
		switch k {
		case keyUp:
			if d.selected > 0 {
				d.selected--
				d.viewStart = adjustViewport(d.selected, d.viewStart, d.rows)
				d.draw()
			}
		case keyDown:
			if d.selected < len(d.items)-1 {
				d.selected++
				d.viewStart = adjustViewport(d.selected, d.viewStart, d.rows)
				d.draw()
			}
		case keyEnter, keyTab:
			return d.items[d.selected].Value, nil
		case keyEsc, keyCtrlC:
			return "", nil
		}
	}
}

type dropdown struct {
	items     []Item
	selected  int
	viewStart int
	rows      int
	tty       *os.File
}

const (
	csiSaveCursor    = "\033[s"
	csiRestoreCursor = "\033[u"
	csiClearBelow    = "\033[J"
	csiHideCursor    = "\033[?25l"
	csiShowCursor    = "\033[?25h"
	csiReverse       = "\033[7m"
	csiDim           = "\033[2m"
	csiReset         = "\033[0m"
	csiFaint         = "\033[90m"
	csiCyan          = "\033[36m"
	csiBoldCyan      = "\033[1;36m"
)

// barChar is the left-edge gutter rendered cyan on every row — a visual
// marker so the dropdown is unmistakably vbas, not zsh's default
// menuselect or some other completion plugin.
const barChar = "▍"

func (d *dropdown) init() {
	io.WriteString(d.tty, csiSaveCursor+csiHideCursor)
}

func (d *dropdown) cleanup() {
	io.WriteString(d.tty, csiRestoreCursor+csiClearBelow+csiShowCursor)
}

const (
	valueWidth = 22
	descWidth  = 56
)

func (d *dropdown) draw() {
	var b strings.Builder
	b.WriteString(csiRestoreCursor)
	b.WriteString(csiClearBelow)

	for i := 0; i < d.rows; i++ {
		idx := d.viewStart + i
		if idx >= len(d.items) {
			break
		}
		b.WriteString("\r\n")
		// Left-edge bar is drawn outside the reverse-video region so it
		// stays the same color whether the row is selected or not.
		b.WriteString(csiCyan)
		b.WriteString(barChar)
		b.WriteString(csiReset)
		if idx == d.selected {
			b.WriteString(csiReverse)
		}
		b.WriteString(formatRow(d.items[idx], idx == d.selected))
		b.WriteString(csiReset)
	}

	// Footer with vbas tag, key hints, and position counter.
	b.WriteString("\r\n")
	b.WriteString(csiFaint)
	b.WriteString("─── ")
	b.WriteString(csiReset)
	b.WriteString(csiBoldCyan)
	b.WriteString("vbas")
	b.WriteString(csiReset)
	b.WriteString(csiFaint)
	fmt.Fprintf(&b, " ─── ↑↓ select · ⏎ accept · esc cancel · %d/%d",
		d.selected+1, len(d.items))
	b.WriteString(csiReset)

	io.WriteString(d.tty, b.String())
}

func formatRow(item Item, selected bool) string {
	val := truncate(item.Value, valueWidth)
	desc := truncate(item.Description, descWidth)
	if selected {
		return fmt.Sprintf(" %-*s  %s ", valueWidth, val, desc)
	}
	if desc == "" {
		return fmt.Sprintf(" %-*s ", valueWidth, val)
	}
	return fmt.Sprintf(" %-*s  "+csiFaint+"%s"+csiReset, valueWidth, val, desc)
}

// truncate returns s with at most w runes, appending "…" if truncated.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= w {
		return s
	}
	out := make([]rune, 0, w)
	count := 0
	for _, r := range s {
		if count >= w-1 {
			break
		}
		out = append(out, r)
		count++
	}
	out = append(out, '…')
	return string(out)
}

// adjustViewport keeps the selected index inside the visible window,
// returning the new viewStart.
func adjustViewport(selected, viewStart, rows int) int {
	if selected < viewStart {
		return selected
	}
	if selected >= viewStart+rows {
		return selected - rows + 1
	}
	return viewStart
}

type key int

const (
	keyUnknown key = iota
	keyUp
	keyDown
	keyLeft
	keyRight
	keyEnter
	keyTab
	keyEsc
	keyCtrlC
)

// readKey reads one keystroke from the tty. Escape sequences for arrow
// keys are decoded; bare Esc is disambiguated from CSI prefixes by a
// short read deadline.
func (d *dropdown) readKey() (key, error) {
	buf := make([]byte, 1)
	n, err := d.tty.Read(buf)
	if err != nil || n == 0 {
		return keyUnknown, err
	}
	switch buf[0] {
	case '\r', '\n':
		return keyEnter, nil
	case '\t':
		return keyTab, nil
	case 0x03:
		return keyCtrlC, nil
	case 0x1b:
		return d.readEscapeSeq()
	}
	return keyUnknown, nil
}

func (d *dropdown) readEscapeSeq() (key, error) {
	// 50ms is the standard heuristic to disambiguate bare Esc from CSI prefix.
	_ = d.tty.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	defer d.tty.SetReadDeadline(time.Time{})

	var buf [2]byte
	if _, err := d.tty.Read(buf[:1]); err != nil {
		return keyEsc, nil
	}
	if _, err := d.tty.Read(buf[1:2]); err != nil {
		return keyEsc, nil
	}
	return parseEscape(buf[0], buf[1]), nil
}

// parseEscape decodes the two bytes following Esc into a key. Both CSI
// ('[') and SS3 ('O') prefixes precede arrow keys; SS3 is sent when the
// terminal is in application cursor-key mode (DECCKM), which zsh's ZLE
// enables by default — that's why the M2-as-shipped version ignored
// arrow keys inside zsh.
func parseEscape(prefix, code byte) key {
	if prefix != '[' && prefix != 'O' {
		return keyEsc
	}
	switch code {
	case 'A':
		return keyUp
	case 'B':
		return keyDown
	case 'C':
		return keyRight
	case 'D':
		return keyLeft
	}
	return keyUnknown
}
