package permission

import (
	"bufio"
	"fmt"
	"strings"

	"agent-harness/Squib/internal/ui"
)

// Gate enforces user confirmation before executing potentially destructive tools.
type Gate struct {
	scanner *bufio.Scanner
	always  map[string]bool
}

func NewGate(scanner *bufio.Scanner) *Gate {
	return &Gate{
		scanner: scanner,
		always:  make(map[string]bool),
	}
}

// Ask prompts the user if the tool is not pre-approved or safe.
func (g *Gate) Ask(toolName string, rawArgs string) bool {
	if g.always[toolName] {
		return true
	}

	// Safe read-only tools can run automatically
	if toolName == "read_file" || toolName == "devops_status" {
		return true
	}

	fmt.Println(ui.Warn("\n⚠️  [PERMISSION GATE] The agent wants to execute:"))
	fmt.Printf("   Tool:      %s\n", ui.Accent(toolName))
	fmt.Printf("   Arguments: %s\n", strings.TrimSpace(rawArgs))
	fmt.Print(ui.Warn("   Allow execution? [y=Yes / n=No / a=Always allow this tool]: "))

	if !g.scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(g.scanner.Text()))

	switch answer {
	case "y", "yes", "si", "s":
		return true
	case "a", "always":
		g.always[toolName] = true
		fmt.Printf("   (Tool %q will be allowed automatically for the rest of this session)\n", toolName)
		return true
	default:
		fmt.Println(ui.Error("   ❌ Execution denied by user."))
		return false
	}
}
