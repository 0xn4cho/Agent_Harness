//go:build windows

package ui

import (
	"os"
	"syscall"
)

// enableVirtualTerminal turns on ANSI escape-code processing in the Windows
// console (ENABLE_VIRTUAL_TERMINAL_PROCESSING). Windows Terminal already has
// it on, but the classic console (conhost) needs it to be enabled explicitly.
func enableVirtualTerminal() bool {
	const enableVirtualTerminalProcessing = 0x0004

	handle := syscall.Handle(os.Stdout.Fd())
	var mode uint32
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}

	setConsoleMode := syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")
	ok, _, _ := setConsoleMode.Call(uintptr(handle), uintptr(mode|enableVirtualTerminalProcessing))
	return ok != 0
}
