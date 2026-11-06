package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	ansiRegex = regexp.MustCompile(`\x1b(\[[0-9;?]*[a-zA-Z]|\][^\x07\x1b]*(\x07|\x1b\\)|[PX^_][^\x1b]*\x1b\\|.)`)

	sensitivePatterns = []string{
		".ssh",
		".kube",
		".aws",
		".azure",
		".gcp",
		".docker",
		".gnupg",
		".env",
		"id_rsa",
		"id_ecdsa",
		"id_ed25519",
		"credentials",
		"authorized_keys",
		"known_hosts",
		"kubeconfig",
		"config.json",
	}

	sensitiveEnvSubstrings = []string{
		"GROQ",
		"OPENAI",
		"ANTHROPIC",
		"API_KEY",
		"SECRET",
		"TOKEN",
		"PASSWORD",
		"PASSWD",
		"CREDENTIAL",
		"AUTH",
	}

	drivePathRegex = regexp.MustCompile(`(?i)(^|[\s"'=])[a-z]:[\\/]`)

	evalSymlinks = filepath.EvalSymlinks
)

type CommandRisk string

const (
	RiskLow     CommandRisk = "low"
	RiskHigh    CommandRisk = "high"
	RiskBlocked CommandRisk = "blocked"
)

type CommandAssessment struct {
	Risk    CommandRisk
	Reasons []string
}

func SanitizeTerminal(s string) string {
	cleaned := ansiRegex.ReplaceAllString(s, "")

	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... [truncated]"
}

func ValidateSafePath(targetPath string) (string, error) {
	if strings.TrimSpace(targetPath) == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace directory: %w", err)
	}

	realCwd, err := evalSymlinks(absCwd)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate workspace symlinks: %w", err)
	}
	realCwd = normalizeVolumePath(realCwd)

	var absTarget string
	if filepath.IsAbs(targetPath) {
		absTarget = filepath.Clean(targetPath)
	} else {
		absTarget = filepath.Clean(filepath.Join(absCwd, targetPath))
	}

	if !isSubPath(absCwd, absTarget) {
		return "", fmt.Errorf("access denied: path %q is outside the allowed workspace directory (%s)", targetPath, absCwd)
	}

	if err := checkSensitivePatterns(absTarget); err != nil {
		return "", err
	}

	realTarget, err := resolvePathWithSymlinks(absTarget)
	if err != nil {
		return "", err
	}
	realTarget = normalizeVolumePath(realTarget)

	if !isSubPath(realCwd, realTarget) {
		return "", fmt.Errorf("access denied: path %q resolves outside the allowed workspace directory via symlink or junction (%s)", targetPath, realTarget)
	}

	if err := checkSensitivePatterns(realTarget); err != nil {
		return "", err
	}

	return realTarget, nil
}

func resolvePathWithSymlinks(absPath string) (string, error) {
	curr := filepath.Clean(absPath)
	var tail []string

	for {
		info, lerr := os.Lstat(curr)
		if lerr == nil {
			resolved, err := evalSymlinks(curr)
			if err != nil {
				if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeIrregular != 0 {
					return "", fmt.Errorf("access denied: path %q contains a broken or unresolvable symlink/junction", absPath)
				}
				return "", fmt.Errorf("access denied: unable to resolve path %q: %w", absPath, err)
			}

			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return filepath.Clean(resolved), nil
		}

		if !os.IsNotExist(lerr) {
			return "", fmt.Errorf("failed to inspect path %q: %w", curr, lerr)
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", fmt.Errorf("access denied: root path does not exist for %q", absPath)
		}
		tail = append(tail, filepath.Base(curr))
		curr = parent
	}
}

func normalizeVolumePath(p string) string {
	p = filepath.Clean(p)
	p = strings.TrimPrefix(p, `\\?\`)
	return p
}

func isSubPath(baseDir, target string) bool {
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if !strings.EqualFold(filepath.VolumeName(baseDir), filepath.VolumeName(target)) {
		return false
	}
	return true
}

func checkSensitivePatterns(target string) error {
	lowerTarget := strings.ToLower(filepath.ToSlash(target))
	base := strings.ToLower(filepath.Base(target))

	if strings.HasPrefix(base, ".env") {
		return fmt.Errorf("access denied: environment file %q is blocked for security", base)
	}

	for _, pattern := range sensitivePatterns {
		lowerPattern := strings.ToLower(pattern)
		if base == lowerPattern ||
			strings.Contains(lowerTarget, "/"+lowerPattern+"/") ||
			strings.HasSuffix(lowerTarget, "/"+lowerPattern) {
			return fmt.Errorf("access denied: sensitive path or file pattern %q is blocked for security", pattern)
		}
	}
	return nil
}

func ClassifyCommand(cmdStr string) CommandAssessment {
	trimmed := strings.TrimSpace(cmdStr)
	if trimmed == "" {
		return CommandAssessment{Risk: RiskLow}
	}

	lower := strings.ToLower(trimmed)

	blockedChecks := []struct {
		pattern string
		reason  string
	}{
		{"-encodedcommand", "obfuscated PowerShell -EncodedCommand is forbidden"},
		{"-enc ", "obfuscated PowerShell -enc flag is forbidden"},
		{"frombase64string", "base64 payload decoding in shell is forbidden"},
		{"invoke-expression", "dynamic code evaluation (Invoke-Expression) is forbidden"},
		{".ssh", "direct shell access to .ssh credentials is forbidden"},
		{"id_rsa", "direct shell access to SSH private keys is forbidden"},
		{"id_ed25519", "direct shell access to SSH private keys is forbidden"},
		{".aws/credentials", "direct shell access to AWS credentials is forbidden"},
		{`.aws\credentials`, "direct shell access to AWS credentials is forbidden"},
		{".gnupg", "direct shell access to GPG keys is forbidden"},
		{"mimikatz", "credential dumping tool is forbidden"},
		{"sekurlsa", "credential dumping command is forbidden"},
	}

	for _, bc := range blockedChecks {
		if strings.Contains(lower, bc.pattern) {
			return CommandAssessment{
				Risk:    RiskBlocked,
				Reasons: []string{bc.reason},
			}
		}
	}

	var reasons []string

	if strings.ContainsAny(trimmed, ";|&><`\r\n") || strings.Contains(trimmed, "$(") || strings.Contains(trimmed, "@(") {
		reasons = append(reasons, "uses shell chaining, pipelines, redirection, or subexpressions")
	}

	if hasParentTraversal(trimmed) ||
		drivePathRegex.MatchString(trimmed) ||
		strings.Contains(trimmed, `\\`) ||
		hasUnixRootPathToken(trimmed) {
		reasons = append(reasons, "references absolute paths or parent directory traversal (..)")
	}

	envOrHomeIndicators := []string{
		"$home", "~", "$profile", "$env:", "%userprofile%", "%appdata%", "%localappdata%",
		"hklm:", "hkcu:", "registry::", "cert:",
	}
	for _, ind := range envOrHomeIndicators {
		if strings.Contains(lower, ind) {
			reasons = append(reasons, "accesses $HOME, environment variables, or Windows registry/certificate stores")
			break
		}
	}

	networkIndicators := []string{
		"invoke-webrequest", "invoke-restmethod", "iwr ", "irm ", "curl ", "wget ",
		"start-bitstransfer", "net.webclient", "http://", "https://", "ssh ", "scp ",
	}
	for _, netInd := range networkIndicators {
		if strings.Contains(lower, netInd) {
			reasons = append(reasons, "performs network or remote communication")
			break
		}
	}

	if !isAllowlistedInspectionCommand(trimmed) {
		reasons = append(reasons, "not in read-only workspace inspection allowlist (requires full shell approval)")
	}

	if len(reasons) > 0 {
		return CommandAssessment{
			Risk:    RiskHigh,
			Reasons: reasons,
		}
	}

	return CommandAssessment{
		Risk:    RiskLow,
		Reasons: []string{"allowlisted read-only workspace inspection"},
	}
}

func hasParentTraversal(cmd string) bool {
	cleaned := strings.ReplaceAll(cmd, "...", "")
	return strings.Contains(cleaned, "..")
}

func hasUnixRootPathToken(cmd string) bool {
	for _, tok := range strings.Fields(cmd) {
		clean := strings.Trim(tok, `"'`)
		if strings.HasPrefix(clean, "/") {
			return true
		}
	}
	return false
}

func isAllowlistedInspectionCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}

	verb := strings.ToLower(fields[0])
	args := fields[1:]

	switch verb {
	case "pwd", "get-location", "gl", "whoami", "hostname":
		return len(args) == 0

	case "dir", "ls", "get-childitem", "gci":
		for _, a := range args {
			clean := strings.Trim(a, `"'`)
			if strings.HasPrefix(clean, "-") {
				lowerFlag := strings.ToLower(clean)
				if lowerFlag != "-Force" && lowerFlag != "-force" && lowerFlag != "-name" &&
					lowerFlag != "-recurse" && lowerFlag != "-r" && lowerFlag != "-l" && lowerFlag != "-a" && lowerFlag != "-la" {
					return false
				}
				continue
			}
			if _, err := ValidateSafePath(clean); err != nil {
				return false
			}
		}
		return true

	case "git":
		if len(args) == 0 {
			return false
		}
		sub := strings.ToLower(args[0])
		switch sub {
		case "status", "diff", "log", "branch", "show", "rev-parse":
			for _, a := range args[1:] {
				lowerArg := strings.ToLower(a)
				if strings.HasPrefix(lowerArg, "--output") ||
					strings.HasPrefix(lowerArg, "--exec") ||
					strings.HasPrefix(lowerArg, "--git-dir") ||
					strings.HasPrefix(lowerArg, "--work-tree") ||
					lowerArg == "-c" {
					return false
				}
			}
			return true
		default:
			return false
		}

	case "go":
		if len(args) == 0 {
			return false
		}
		sub := strings.ToLower(args[0])
		switch sub {
		case "version", "list", "vet", "doc":
			return true
		default:
			return false
		}

	case "docker":
		if len(args) == 0 {
			return false
		}
		sub := strings.ToLower(args[0])
		switch sub {
		case "ps", "images", "version":
			return true
		default:
			return false
		}

	default:
		return false
	}
}

func SanitizedEnviron() []string {
	var env []string
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 0 {
			continue
		}
		key := strings.ToUpper(parts[0])

		isSensitive := false
		for _, sub := range sensitiveEnvSubstrings {
			if strings.Contains(key, sub) {
				isSensitive = true
				break
			}
		}

		if !isSensitive {
			env = append(env, e)
		}
	}
	return env
}
