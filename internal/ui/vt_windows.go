package ui

import (
	"os"
	"syscall"
	"unsafe"
)

type keyEventRecord struct {
	bKeyDown          int32
	wRepeatCount      uint16
	wVirtualKeyCode   uint16
	wVirtualScanCode  uint16
	unicodeChar       uint16
	dwControlKeyState uint32
}

type inputRecord struct {
	eventType uint16
	_         uint16
	keyEvent  keyEventRecord
}

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

func waitForFirstKey() bool {
	const keyEventType = 0x0001

	stdinHandle := syscall.Handle(os.Stdin.Fd())
	var mode uint32
	if err := syscall.GetConsoleMode(stdinHandle, &mode); err != nil {
		return false
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	peekConsoleInputW := kernel32.NewProc("PeekConsoleInputW")
	readConsoleInputW := kernel32.NewProc("ReadConsoleInputW")

	for {
		event, err := syscall.WaitForSingleObject(stdinHandle, syscall.INFINITE)
		if err != nil || event != syscall.WAIT_OBJECT_0 {
			return false
		}

		var rec inputRecord
		var readCount uint32
		ok, _, _ := peekConsoleInputW.Call(
			uintptr(stdinHandle),
			uintptr(unsafe.Pointer(&rec)),
			1,
			uintptr(unsafe.Pointer(&readCount)),
		)
		if ok == 0 {
			return false
		}
		if readCount == 0 {
			continue
		}

		if rec.eventType == keyEventType && rec.keyEvent.bKeyDown != 0 && rec.keyEvent.unicodeChar != 0 {
			return true
		}

		var discarded uint32
		ok, _, _ = readConsoleInputW.Call(
			uintptr(stdinHandle),
			uintptr(unsafe.Pointer(&rec)),
			1,
			uintptr(unsafe.Pointer(&discarded)),
		)
		if ok == 0 {
			return false
		}
	}
}
