package permission

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"agent-harness/Squib/internal/security"
	"agent-harness/Squib/internal/ui"
)

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

func (g *Gate) Ask(toolName string, rawArgs string) bool {
	if g.always[toolName] {
		return true
	}

	sanitizedArgs := security.SanitizeTerminal(strings.TrimSpace(rawArgs))

	if toolName == "bash" {
		var parsed map[string]any
		cmdStr := ""
		if err := json.Unmarshal([]byte(rawArgs), &parsed); err == nil {
			cmdStr, _ = parsed["command"].(string)
		} else {
			cmdStr = rawArgs
		}

		assessment := security.ClassifyCommand(cmdStr)
		if assessment.Risk == security.RiskBlocked {
			fmt.Println(ui.Error("\n🛑 [SECURITY GATE] Command blocked by security policy:"))
			fmt.Printf("   Command:   %s\n", sanitizedArgs)
			fmt.Printf("   Reason:    %s\n", strings.Join(assessment.Reasons, "; "))
			return false
		}

		if assessment.Risk == security.RiskHigh {
			fmt.Println(ui.Error("\n🚨 [REINFORCED SHELL APPROVAL] Full shell access requested:"))
			fmt.Printf("   Tool:      %s (elevated risk — outside inspection allowlist)\n", ui.Accent(toolName))
			fmt.Printf("   Arguments: %s\n", sanitizedArgs)
			for _, r := range assessment.Reasons {
				fmt.Printf("   • Risk:    %s\n", ui.Warn(r))
			}
			fmt.Print(ui.Warn("   Type 'yes' (or 'si') in full to approve full-shell execution [yes/no]: "))

			if !g.scanner.Scan() {
				if err := g.scanner.Err(); err != nil {
					fmt.Println(ui.Error(fmt.Sprintf("   ❌ Input error: %v", err)))
				}
				return false
			}
			answer := strings.ToLower(strings.TrimSpace(g.scanner.Text()))
			if answer == "yes" || answer == "si" {
				return true
			}
			fmt.Println(ui.Error("   ❌ Reinforced approval denied (must type 'yes' or 'si' in full for non-allowlisted shell commands)."))
			return false
		}

		fmt.Println(ui.Warn("\n⚠️  [PERMISSION GATE] Allowlisted low-risk inspection command:"))
		fmt.Printf("   Tool:      %s (low-risk inspection)\n", ui.Accent(toolName))
		fmt.Printf("   Arguments: %s\n", sanitizedArgs)
		fmt.Print(ui.Warn("   Allow execution? [y=Yes / n=No]: "))
	} else {
		fmt.Println(ui.Warn("\n⚠️  [PERMISSION GATE] The agent wants to execute:"))
		fmt.Printf("   Tool:      %s\n", ui.Accent(toolName))
		fmt.Printf("   Arguments: %s\n", sanitizedArgs)
		fmt.Print(ui.Warn("   Allow execution? [y=Yes / n=No / a=Always allow this tool]: "))
	}

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
		if toolName == "bash" || toolName == "write_file" || toolName == "devops_status" {
			fmt.Println(ui.Warn("   ⚠️ Shell commands, file writes, and DevOps status checks require per-operation confirmation for strict isolation."))
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
