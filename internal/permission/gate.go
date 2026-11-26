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

	target := ui.SummarizeToolArgs(toolName, rawArgs)
	if target == "" {
		target = security.SanitizeTerminal(strings.TrimSpace(rawArgs))
	}

	if isExternalFilesystemTool(toolName) {
		fmt.Println()
		fmt.Printf("  %s %s %s %s %s %s\n", ui.Error("│"), ui.Error("Permiso externo"), ui.Dim("·"), ui.Cyan(toolName), ui.Dim("→"), ui.Text(target))
		fmt.Printf("  %s %s %s ", ui.Error("│"), ui.Warn("¿Aprobar? [escribe 'yes' o 'si']"), ui.PromptGlyph())
		if !g.scanner.Scan() {
			return false
		}
		answer := strings.ToLower(strings.TrimSpace(g.scanner.Text()))
		if answer == "yes" || answer == "si" {
			return true
		}
		fmt.Printf("  %s %s\n", ui.Error("│"), ui.Error("Denegado (requiere 'yes' o 'si')"))
		return false
	}

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
			fmt.Println()
			fmt.Printf("  %s %s %s %s %s %s\n", ui.Error("│"), ui.Error("Bloqueado"), ui.Dim("·"), ui.Cyan(toolName), ui.Dim("→"), ui.Text(target))
			fmt.Printf("  %s %s\n", ui.Error("│"), ui.Dim(strings.Join(assessment.Reasons, "; ")))
			return false
		}

		if assessment.Risk == security.RiskHigh {
			fmt.Println()
			fmt.Printf("  %s %s %s %s %s %s\n", ui.Error("│"), ui.Error("Permiso reforzado"), ui.Dim("·"), ui.Cyan(toolName), ui.Dim("→"), ui.Text(target))
			for _, r := range assessment.Reasons {
				fmt.Printf("  %s %s\n", ui.Error("│"), ui.Dim("Riesgo: "+r))
			}
			fmt.Printf("  %s %s %s ", ui.Error("│"), ui.Warn("¿Aprobar? [escribe 'yes' o 'si']"), ui.PromptGlyph())

			if !g.scanner.Scan() {
				if err := g.scanner.Err(); err != nil {
					fmt.Printf("  %s %s\n", ui.Error("│"), ui.Error(fmt.Sprintf("Error de entrada: %v", err)))
				}
				return false
			}
			answer := strings.ToLower(strings.TrimSpace(g.scanner.Text()))
			if answer == "yes" || answer == "si" {
				return true
			}
			fmt.Printf("  %s %s\n", ui.Error("│"), ui.Error("Denegado (requiere 'yes' o 'si')"))
			return false
		}

		fmt.Println()
		fmt.Printf("  %s %s %s %s %s %s\n", ui.Warn("│"), ui.Warn("Permiso"), ui.Dim("·"), ui.Cyan(toolName), ui.Dim("→"), ui.Text(target))
		fmt.Printf("  %s %s %s ", ui.Warn("│"), ui.Dim("¿Aprobar? [y/n]"), ui.PromptGlyph())
	} else {
		fmt.Println()
		fmt.Printf("  %s %s %s %s %s %s\n", ui.Warn("│"), ui.Warn("Permiso"), ui.Dim("·"), ui.Cyan(toolName), ui.Dim("→"), ui.Text(target))
		fmt.Printf("  %s %s %s ", ui.Warn("│"), ui.Dim("¿Aprobar? [y/n/a]"), ui.PromptGlyph())
	}

	if !g.scanner.Scan() {
		if err := g.scanner.Err(); err != nil {
			fmt.Printf("  %s %s\n", ui.Error("│"), ui.Error(fmt.Sprintf("Error de entrada: %v", err)))
		}
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(g.scanner.Text()))

	switch answer {
	case "y", "yes", "si", "s":
		return true
	case "a", "always":
		if toolName == "bash" || toolName == "write_file" || toolName == "devops_status" || isExternalFilesystemTool(toolName) {
			fmt.Printf("  %s %s\n", ui.Warn("│"), ui.Dim("Aprobado solo para esta operación"))
			return true
		}
		g.always[toolName] = true
		fmt.Printf("  %s %s\n", ui.Success("│"), ui.Dim(fmt.Sprintf("Permitido %q durante esta sesión", toolName)))
		return true
	default:
		fmt.Printf("  %s %s\n", ui.Error("│"), ui.Error("Denegado"))
		return false
	}
}

func isExternalFilesystemTool(toolName string) bool {
	switch toolName {
	case "list_directory", "find_files", "copy_path", "move_path":
		return true
	default:
		return false
	}
}
