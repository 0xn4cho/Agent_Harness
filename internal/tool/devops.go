package tool

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"agent-harness/Squib/internal/api"
)

// DevOpsTool provides quick health and status inspection for Docker and Kubernetes.
type DevOpsTool struct{}

func NewDevOpsTool() *DevOpsTool {
	return &DevOpsTool{}
}

func (d *DevOpsTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "devops_status",
		Description: "Inspect current Docker containers and Kubernetes cluster pods.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target": map[string]any{
					"type":        "string",
					"enum":        []string{"docker", "k8s", "all"},
					"description": "Which system to check: docker, k8s, or all.",
				},
			},
			"required": []string{"target"},
		},
	}
}

func (d *DevOpsTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	target, _ := args["target"].(string)
	if target == "" {
		target = "all"
	}

	runCmd := func(command string) string {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("[not running / error: %v]\n%s", err, string(out))
		}
		return string(out)
	}

	var output string
	if target == "docker" || target == "all" {
		output += "=== DOCKER STATUS ===\n"
		output += runCmd("docker ps --format 'table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Names}}'")
		output += "\n"
	}
	if target == "k8s" || target == "all" {
		output += "=== KUBERNETES STATUS ===\n"
		output += runCmd("kubectl get pods -A")
		output += "\n"
	}

	return output, nil
}
