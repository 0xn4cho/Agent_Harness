//go:build !windows

package ui

// enableVirtualTerminal is a no-op on Linux/macOS: their terminals understand
// ANSI escape codes natively.
func enableVirtualTerminal() bool { return true }
