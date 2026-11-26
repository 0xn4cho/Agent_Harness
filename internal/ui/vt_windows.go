package ui

import (
	"os"
	"syscall"
	"unsafe"
)

type coord struct {
	x int16
	y int16
}

type smallRect struct {
	left   int16
	top    int16
	right  int16
	bottom int16
}

type consoleScreenBufferInfo struct {
	size              coord
	cursorPosition    coord
	attributes        uint16
	window            smallRect
	maximumWindowSize coord
}

func enableVirtualTerminal() bool {
	const (
		enableVirtualTerminalProcessing = 0x0004
		utf8CodePage                    = 65001
	)

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	_, _, _ = kernel32.NewProc("SetConsoleOutputCP").Call(uintptr(utf8CodePage))
	_, _, _ = kernel32.NewProc("SetConsoleCP").Call(uintptr(utf8CodePage))

	handle := syscall.Handle(os.Stdout.Fd())
	var mode uint32
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}

	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	ok, _, _ := setConsoleMode.Call(uintptr(handle), uintptr(mode|enableVirtualTerminalProcessing))
	return ok != 0
}

func TermWidth() int {
	handle := syscall.Handle(os.Stdout.Fd())
	getConsoleScreenBufferInfo := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleScreenBufferInfo")
	var info consoleScreenBufferInfo
	ok, _, _ := getConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return 0
	}
	w := int(info.window.right - info.window.left + 1)
	if w < 0 {
		return 0
	}
	return w
}
