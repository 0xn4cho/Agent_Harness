//go:build !windows

package ui

func enableVirtualTerminal() bool { return true }

func waitForFirstKey() bool { return false }
