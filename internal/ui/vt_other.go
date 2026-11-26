//go:build !windows

package ui

import (
	"os"
	"strconv"
	"strings"
)

func enableVirtualTerminal() bool { return true }

func TermWidth() int {
	info, err := os.Stdout.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return 0
	}
	if cols, err := strconv.Atoi(strings.TrimSpace(os.Getenv("COLUMNS"))); err == nil && cols > 0 {
		return cols
	}
	return 80
}
