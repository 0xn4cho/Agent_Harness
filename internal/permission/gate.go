package permission

import (
	"bufio"
	"fmt"
	"strings"

	"agent-harness/Squib/internal/security"
	"agent-harness/Squib/internal/ui"
)

// Gate enforces user confirmation before executing potentially destructive or sensitive tools.
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

// Ask prompts the user for confirmation.
func (g *Gate) Ask(toolName string, rawArgs string) bool {
	if g.always[toolName] {
		return true
	}

	// Sanitize raw arguments before displaying them to prevent ANSI injection
	sanitizedArgs := security.SanitizeTerminal(strings.TrimSpace(rawArgs))

	fmt.Println(ui.Warn("\n⚠️  [PERMISSION GATE] The agent wants to execute:"))
	fmt.Printf("   Tool:      %s\n", ui.Accent(toolName))
	fmt.Printf("   Arguments: %s\n", sanitizedArgs)
	fmt.Print(ui.Warn("   Allow execution? [y=Yes / n=No / a=Always allow this tool]: "))

	if !g.scanner.Scan() {
		if err := g.scanner.Err(); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("   ❌ Input error: %v", err)))
		}
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(g.scanner.Text()))

	switch answer {
	case "y", "yes", "si", "s":
		return true
	case "a", "always":
		// Disallow blanket "always" approval for high-risk / arbitrary execution tools
		if toolName == "bash" || toolName == "write_file" {
			fmt.Println(ui.Warn("   ⚠️ Arbitrary shell commands and file writes cannot be permanently auto-approved for security."))
			fmt.Println(ui.Dim("   (Approving this execution once)"))
			return true
		}
		g.always[toolName] = true
		fmt.Printf("   (Tool %q will be allowed automatically for the rest of this session)\n", toolName)
		return true
	default:
		fmt.Println(ui.Error("   ❌ Execution denied by user."))
		return false
	}
}
