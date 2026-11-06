package tool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/security"
)

const (
	devopsCmdTimeout = 15 * time.Second
	maxDevopsOutput  = 64 * 1024
)

type DevOpsTool struct {
	KubeconfigPath string
}

func NewDevOpsTool() *DevOpsTool {
	return &DevOpsTool{
		KubeconfigPath: strings.TrimSpace(os.Getenv("SQUIB_KUBECONFIG")),
	}
}

func (d *DevOpsTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "devops_status",
		Description: "Inspect current Docker containers and Kubernetes cluster pods using an explicit controlled kubeconfig.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target": map[string]any{
					"type":        "string",
					"enum":        []string{"docker", "k8s", "all"},
					"description": "Which system to check: docker, k8s, or all.",
				},
				"kubeconfig": map[string]any{
					"type":        "string",
					"description": "Optional workspace-relative path to an explicit controlled kubeconfig file for Kubernetes checks.",
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

	switch target {
	case "docker", "k8s", "all":
	default:
		return "", fmt.Errorf("invalid 'target' %q: must be one of 'docker', 'k8s', or 'all'", target)
	}

	runExec := func(bin string, extraEnv []string, cmdArgs ...string) string {
		cmdCtx, cancel := context.WithTimeout(ctx, devopsCmdTimeout)
		defer cancel()

		cmd := exec.CommandContext(cmdCtx, bin, cmdArgs...)

		var cleanEnv []string
		for _, e := range security.SanitizedEnviron() {
			if !strings.HasPrefix(strings.ToUpper(e), "KUBECONFIG=") {
				cleanEnv = append(cleanEnv, e)
			}
		}
		cmd.Env = append(cleanEnv, extraEnv...)

		if cwd, err := os.Getwd(); err == nil {
			cmd.Dir = cwd
		}

		out, err := runWithLimit(cmdCtx, cmd, maxDevopsOutput)
		if cmdCtx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("[command timed out after %v]\n%s", devopsCmdTimeout, out)
		}
		if err != nil {
			return fmt.Sprintf("[not running / error: %v]\n%s", err, out)
		}
		return out
	}

	var output string
	if target == "docker" || target == "all" {
		output += "=== DOCKER STATUS ===\n"
		output += runExec("docker", nil, "ps", "--format", "table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Names}}")
		output += "\n"
	}
	if target == "k8s" || target == "all" {
		output += "=== KUBERNETES STATUS ===\n"
		kubeconfig, err := d.resolveControlledKubeconfig(args)
		if err != nil {
			output += fmt.Sprintf("[k8s isolation: %v]\n", err)
		} else {
			env := []string{"KUBECONFIG=" + kubeconfig}
			output += runExec("kubectl", env, "--kubeconfig", kubeconfig, "get", "pods", "-A")
			output += "\n"
		}
	}

	return output, nil
}

func (d *DevOpsTool) resolveControlledKubeconfig(args map[string]any) (string, error) {
	if rawKC, ok := args["kubeconfig"].(string); ok && strings.TrimSpace(rawKC) != "" {
		return validateWorkspaceKubeconfig(rawKC)
	}

	if d.KubeconfigPath != "" {
		info, err := os.Stat(d.KubeconfigPath)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("configured SQUIB_KUBECONFIG %q is not a valid file", d.KubeconfigPath)
		}
		return d.KubeconfigPath, nil
	}

	if envKC := strings.TrimSpace(os.Getenv("SQUIB_KUBECONFIG")); envKC != "" {
		info, err := os.Stat(envKC)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("configured SQUIB_KUBECONFIG %q is not a valid file", envKC)
		}
		return envKC, nil
	}

	if safe, err := validateWorkspaceKubeconfig("k8s-config.yaml"); err == nil {
		if info, statErr := os.Stat(safe); statErr == nil && !info.IsDir() {
			return safe, nil
		}
	}

	return "", fmt.Errorf("global ~/.kube/config access is disabled for workspace isolation; set SQUIB_KUBECONFIG or pass a workspace 'kubeconfig' path")
}

func validateWorkspaceKubeconfig(targetPath string) (string, error) {
	if strings.TrimSpace(targetPath) == "" {
		return "", fmt.Errorf("kubeconfig path cannot be empty")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	absTarget := targetPath
	if !filepath.IsAbs(absTarget) {
		absTarget = filepath.Join(cwd, absTarget)
	}
	absTarget = filepath.Clean(absTarget)

	parentDir := filepath.Dir(absTarget)
	safeParent, err := security.ValidateSafePath(parentDir)
	if err != nil {
		return "", fmt.Errorf("kubeconfig parent directory is not safe: %w", err)
	}

	candidate := filepath.Join(safeParent, filepath.Base(absTarget))
	if info, lerr := os.Lstat(candidate); lerr == nil {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			return "", fmt.Errorf("kubeconfig contains broken symlink/junction: %w", err)
		}
		if _, err := security.ValidateSafePath(filepath.Dir(resolved)); err != nil {
			return "", fmt.Errorf("kubeconfig symlink/junction escapes workspace: %w", err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("kubeconfig path %q is a directory", targetPath)
		}
		return resolved, nil
	}

	return candidate, nil
}
