package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/permission"
	"agent-harness/Squib/internal/provider"
	"agent-harness/Squib/internal/tool"
	"agent-harness/Squib/internal/ui"
)

const systemPrompt = `You are Squib, an expert coding and DevOps AI agent running in a terminal on Windows.
You have tools to inspect code, read/write files, check Docker/Kubernetes, and run PowerShell commands.

Guidelines:
1. Operating System: You are running on Windows. For shell commands, use PowerShell syntax. Never use Linux commands like 'chmod' or 'chown' (permissions on Windows NTFS do not use them).
2. File Operations: Always prefer 'read_file' and 'write_file' tools over PowerShell redirection or echo commands. Pay close attention to exact folder names (e.g. 'examples' plural, not 'example').
3. DevOps: Inspect existing files and container/cluster state before creating or modifying configurations.
4. Keep responses direct, helpful, and concise.`

func main() {
	model := os.Getenv("GROQ_MODEL")
	if model == "" {
		model = "openai/gpt-oss-120b"
	}

	// 0. Banner de bienvenida (pulpo + logo SQUIB)
	fmt.Print(ui.Banner(model))

	apiKey := strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, ui.Error("\nError: GROQ_API_KEY environment variable is not set."))
		fmt.Fprintln(os.Stderr, ui.Dim("Set it in PowerShell: $env:GROQ_API_KEY='gsk_...'"))
		os.Exit(1)
	}

	// 1. Inicializar el Provider (Groq)
	p := provider.NewGroq(apiKey, model)

	// 2. Registrar herramientas de Código y DevOps
	reg := tool.NewRegistry()
	reg.Register(tool.NewBashTool())
	reg.Register(tool.NewReadFileTool())
	reg.Register(tool.NewWriteFileTool())
	reg.Register(tool.NewDevOpsTool())

	// 3. Crear el Permission Gate
	scanner := bufio.NewScanner(os.Stdin)
	gate := permission.NewGate(scanner)

	messages := []api.Message{{Role: "system", Content: systemPrompt}}

	for {
		fmt.Print("\n" + ui.Prompt())
		if !scanner.Scan() {
			return
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println(ui.Accent("¡Hasta luego! 🐙"))
			return
		}

		messages = append(messages, api.Message{Role: "user", Content: input})
		messages = agentLoop(context.Background(), p, reg, gate, messages)
	}
}

func agentLoop(ctx context.Context, p provider.Provider, reg *tool.Registry, gate *permission.Gate, messages []api.Message) []api.Message {
	const maxRounds = 10
	toolDefs := reg.Definitions()

	for round := 0; round < maxRounds; round++ {
		respMsg, err := p.Send(ctx, messages, toolDefs)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("API error: %v", err)))
			return messages
		}

		messages = append(messages, *respMsg)

		if text := visibleContent(respMsg.Content); text != "" {
			fmt.Println(text)
		}

		if len(respMsg.ToolCalls) == 0 {
			return messages
		}

		for _, tc := range respMsg.ToolCalls {
			fmt.Println(ui.ToolCall(tc.Name, tc.Arguments))
			toolObj, exists := reg.Get(tc.Name)
			if !exists {
				messages = append(messages, api.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("unknown tool: %s", tc.Name),
				})
				continue
			}

			// Chapter 02: The Permission Gate
			if !gate.Ask(tc.Name, tc.Arguments) {
				messages = append(messages, api.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    "Permission denied by user.",
				})
				continue
			}

			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
				messages = append(messages, api.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("invalid arguments JSON: %v", err),
				})
				continue
			}

			output, err := toolObj.Execute(ctx, args)
			if err != nil {
				output = fmt.Sprintf("[error: %v]", err)
			}
			messages = append(messages, api.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    output,
			})
		}
	}

	fmt.Println(ui.Warn(fmt.Sprintf("Squib se detuvo tras %d rondas de herramientas para evitar un bucle infinito.", maxRounds)))
	return messages
}

func visibleContent(content string) string {
	start := strings.Index(content, "<think>")
	end := strings.Index(content, "</think>")
	switch {
	case start >= 0 && end > start:
		content = content[:start] + content[end+len("</think>"):]
	case start >= 0:
		content = content[:start]
	case end >= 0:
		content = content[end+len("</think>"):]
	}
	return strings.TrimSpace(content)
}
