// Package ui renders an inline dropdown selector below the user's prompt.
//
// The dropdown opens /dev/tty for raw input and ANSI rendering so the
// caller's stdin/stdout/stderr stay clean. The terminal cursor is saved
// on entry (CSI s) and restored on exit (CSI u, then CSI J to clear
// the dropdown rows).
//
// Keys: Up/Down navigate · Tab/Enter accept · Esc/Ctrl-C cancel ·
// printable ASCII filters the list · Backspace removes the last
// filter character.
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
// item's Value, or "" if the user cancels (Esc / Ctrl-C / accepts
// while filtered list is empty).
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

	d := &dropdown{items: items, tty: tty}
	d.refilter()

	d.init()
	defer d.cleanup()

	d.draw()
	for {
		k, ch, err := d.readKey()
		if err != nil {
			return "", nil
		}
		switch k {
		case keyUp:
			if d.selected > 0 {
				d.selected--
				d.viewStart = adjustViewport(d.selected, d.viewStart, MaxRows)
				d.draw()
			}
		case keyDown:
			if d.selected < len(d.filtered)-1 {
				d.selected++
				d.viewStart = adjustViewport(d.selected, d.viewStart, MaxRows)
				d.draw()
			}
		case keyEnter, keyTab:
			if len(d.filtered) == 0 {
				return "", nil
			}
			return d.items[d.filtered[d.selected]].Value, nil
		case keyEsc, keyCtrlC:
			return "", nil
		case keyChar:
			d.query += string(ch)
			d.refilter()
			d.draw()
		case keyBackspace:
			if d.query != "" {
				d.query = d.query[:len(d.query)-1]
				d.refilter()
				d.draw()
			}
		}
	}
}

type dropdown struct {
	items     []Item
	filtered  []int  // indices into items matching the current query
	query     string
	selected  int    // index into filtered
	viewStart int
	tty       *os.File
}

// refilter rebuilds d.filtered from d.items and d.query (case-insensitive
// substring match against Value or Description). Resets selected/viewStart
// to the top of the new filtered list — when the user narrows or widens
// the query, starting at the top is more useful than carrying the old
// position over.
func (d *dropdown) refilter() {
	d.filtered = d.filtered[:0]
	q := strings.ToLower(d.query)
	for i, it := range d.items {
		if q == "" ||
			strings.Contains(strings.ToLower(it.Value), q) ||
			strings.Contains(strings.ToLower(it.Description), q) {
			d.filtered = append(d.filtered, i)
		}
	}
	d.selected = 0
	d.viewStart = 0
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

	rows := len(d.filtered)
	if rows > MaxRows {
		rows = MaxRows
	}
	for i := 0; i < rows; i++ {
		idx := d.viewStart + i
		if idx >= len(d.filtered) {
			break
		}
		item := d.items[d.filtered[idx]]
		b.WriteString("\r\n")
		// Bar drawn outside the reverse-video region so its color is
		// stable regardless of selection.
		b.WriteString(csiCyan)
		b.WriteString(barChar)
		b.WriteString(csiReset)
		if idx == d.selected {
			b.WriteString(csiReverse)
		}
		b.WriteString(formatRow(item, idx == d.selected))
		b.WriteString(csiReset)
	}

	if len(d.filtered) == 0 {
		b.WriteString("\r\n")
		b.WriteString(csiFaint)
		b.WriteString("  (no matches)")
		b.WriteString(csiReset)
	}

	// Footer: ─── vbas [· "query"] ─── hints · X/N
	b.WriteString("\r\n")
	b.WriteString(csiFaint)
	b.WriteString("─── ")
	b.WriteString(csiReset)
	b.WriteString(csiBoldCyan)
	b.WriteString("vbas")
	b.WriteString(csiReset)

	if d.query != "" {
		b.WriteString(csiFaint)
		b.WriteString(" · ")
		b.WriteString(csiReset)
		b.WriteString(csiCyan)
		b.WriteString(fmt.Sprintf("%q", d.query))
		b.WriteString(csiReset)
	}

	b.WriteString(csiFaint)
	if len(d.filtered) > 0 {
		fmt.Fprintf(&b, " ─── ↑↓ select · ⏎ accept · esc cancel · %d/%d",
			d.selected+1, len(d.filtered))
	} else {
		b.WriteString(" ─── type to filter · ⌫ backspace · esc cancel")
	}
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
	keyChar
	keyBackspace
)

// readKey reads one keystroke from the tty and returns (key, char, err).
// char is non-zero only when key == keyChar (a printable ASCII byte).
// Multi-byte UTF-8 input is currently treated as keyUnknown — fine for
// command-name filtering since CLI names are ASCII.
func (d *dropdown) readKey() (key, byte, error) {
	buf := make([]byte, 1)
	n, err := d.tty.Read(buf)
	if err != nil || n == 0 {
		return keyUnknown, 0, err
	}
	b := buf[0]
	switch b {
	case '\r', '\n':
		return keyEnter, 0, nil
	case '\t':
		return keyTab, 0, nil
	case 0x03:
		return keyCtrlC, 0, nil
	case 0x08, 0x7F:
		// 0x08 = Ctrl-H / classic BS; 0x7F = DEL, sent by most modern
		// terminals when Backspace is pressed.
		return keyBackspace, 0, nil
	case 0x1b:
		k, err := d.readEscapeSeq()
		return k, 0, err
	}
	if b >= 0x20 && b <= 0x7E {
		return keyChar, b, nil
	}
	return keyUnknown, 0, nil
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
