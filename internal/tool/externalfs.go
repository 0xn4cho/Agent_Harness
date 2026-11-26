package tool

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/security"
)

const (
	maxExternalResults = 500
	copyBufferSize     = 1024 * 1024
)

type ExternalFileTool struct{ name, description string }

func NewListDirectoryTool() *ExternalFileTool {
	return &ExternalFileTool{name: "list_directory", description: "List entries in a user-selected directory, including directories outside the workspace. Always requires explicit per-operation approval."}
}

func NewFindFilesTool() *ExternalFileTool {
	return &ExternalFileTool{name: "find_files", description: "Recursively find files and directories by filename pattern in a user-selected directory, including outside the workspace. Always requires explicit per-operation approval."}
}

func NewCopyPathTool() *ExternalFileTool {
	return &ExternalFileTool{name: "copy_path", description: "Copy a file or directory between user-selected paths. Refuses to overwrite an existing destination and always requires explicit per-operation approval."}
}

func NewMovePathTool() *ExternalFileTool {
	return &ExternalFileTool{name: "move_path", description: "Move a file or directory between user-selected paths. Refuses to overwrite an existing destination and always requires explicit per-operation approval."}
}

func (t *ExternalFileTool) Definition() api.ToolDef {
	properties := map[string]any{
		"path": map[string]any{"type": "string", "description": "Directory path selected by the user."},
	}
	required := []string{"path"}
	if t.name == "find_files" {
		properties["pattern"] = map[string]any{"type": "string", "description": "Filename pattern such as *.md or *.pdf."}
		required = append(required, "pattern")
	}
	if t.name == "copy_path" || t.name == "move_path" {
		properties = map[string]any{
			"source":      map[string]any{"type": "string", "description": "Existing file or directory to copy or move."},
			"destination": map[string]any{"type": "string", "description": "New destination path. It must not already exist."},
		}
		required = []string{"source", "destination"}
	}
	return api.ToolDef{Name: t.name, Description: t.description, Parameters: map[string]any{"type": "object", "properties": properties, "required": required}}
}

func (t *ExternalFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	switch t.name {
	case "list_directory":
		return listDirectory(args)
	case "find_files":
		return findFiles(args)
	case "copy_path":
		return transferPath(args, false)
	case "move_path":
		return transferPath(args, true)
	default:
		return "", fmt.Errorf("unknown external file operation %q", t.name)
	}
}

func externalArg(args map[string]any, key string) (string, error) {
	v, ok := args[key].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("missing or invalid %q", key)
	}
	return security.ValidateExternalPath(v)
}

func listDirectory(args map[string]any) (string, error) {
	path, err := externalArg(args, "path")
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	if len(entries) > maxExternalResults {
		entries = entries[:maxExternalResults]
	}
	lines := make([]string, 0, len(entries)+1)
	for _, entry := range entries {
		if _, err := security.ValidateExternalPath(filepath.Join(path, entry.Name())); err != nil {
			continue
		}
		kind := "file"
		if entry.IsDir() {
			kind = "dir"
		}
		lines = append(lines, fmt.Sprintf("[%s] %s", kind, entry.Name()))
	}
	if len(entries) == maxExternalResults {
		lines = append(lines, "... [results limited to 500 entries]")
	}
	return strings.Join(lines, "\n"), nil
}

func findFiles(args map[string]any) (string, error) {
	root, err := externalArg(args, "path")
	if err != nil {
		return "", err
	}
	pattern, ok := args["pattern"].(string)
	if !ok || strings.TrimSpace(pattern) == "" {
		return "", fmt.Errorf("missing or invalid %q", "pattern")
	}
	var matches []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if _, err := security.ValidateExternalPath(path); err != nil {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if len(matches) >= maxExternalResults {
			return fs.SkipDir
		}
		matched, matchErr := filepath.Match(pattern, entry.Name())
		if matchErr != nil {
			return matchErr
		}
		if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return "no matches", nil
	}
	if len(matches) == maxExternalResults {
		matches = append(matches, "... [results limited to 500 entries]")
	}
	return strings.Join(matches, "\n"), nil
}

func transferPath(args map[string]any, move bool) (string, error) {
	source, err := externalArg(args, "source")
	if err != nil {
		return "", err
	}
	destination, err := externalArg(args, "destination")
	if err != nil {
		return "", err
	}
	if filepath.Clean(source) == filepath.Clean(destination) {
		return "", fmt.Errorf("source and destination must be different")
	}
	if _, err := os.Lstat(destination); err == nil {
		return "", fmt.Errorf("destination already exists: %s (overwriting is not allowed)", destination)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	info, err := os.Lstat(source)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symbolic links and junctions cannot be copied or moved")
	}
	if err := validateTransferSource(source, info.IsDir()); err != nil {
		return "", err
	}
	if move {
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return "", err
		}
		if err := os.Rename(source, destination); err == nil {
			return "moved " + source + " -> " + destination, nil
		}
	}
	if err := copyTree(source, destination); err != nil {
		return "", err
	}
	if move {
		if err := os.RemoveAll(source); err != nil {
			return "", fmt.Errorf("copied but could not remove source %s: %w", source, err)
		}
		return "moved " + source + " -> " + destination, nil
	}
	return "copied " + source + " -> " + destination, nil
}

func validateTransferSource(source string, isDir bool) error {
	if !isDir {
		return nil
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if _, err := security.ValidateExternalPath(path); err != nil {
			return fmt.Errorf("source contains a protected path: %w", err)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("source contains a symbolic link or junction: %s", path)
		}
		return nil
	})
}

func copyTree(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symbolic links and junctions cannot be copied")
	}
	if !info.IsDir() {
		return copyFile(source, destination, info.Mode())
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links and junctions cannot be copied: %s", path)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, entryInfo.Mode().Perm())
		}
		return copyFile(path, target, entryInfo.Mode())
	})
}

func copyFile(source, destination string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.CopyBuffer(out, in, make([]byte, copyBufferSize))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
