package main

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-harness/Squib/internal/agents"
	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/permission"
	"agent-harness/Squib/internal/provider"
	"agent-harness/Squib/internal/tool"
)

func TestPruneMessages(t *testing.T) {
	system := api.Message{Role: "system", Content: "sys"}

	msgs := []api.Message{system}
	for i := 0; i < 30; i++ {
		msgs = append(msgs, api.Message{
			Role:    "user",
			Content: "hello",
		})
		msgs = append(msgs, api.Message{
			Role:    "assistant",
			Content: "hi",
		})
	}

	pruned := pruneMessages(msgs, 10)
	if len(pruned) > 12 {
		t.Errorf("expected pruned messages length to be <= 12, got %d", len(pruned))
	}

	if pruned[0].Role != "system" {
		t.Errorf("expected first message to remain system, got %s", pruned[0].Role)
	}

	for i, m := range pruned {
		if m.Role == "tool" && i > 0 && pruned[i-1].Role != "assistant" && pruned[i-1].Role != "tool" {
			t.Errorf("tool message at index %d has no preceding assistant/tool call", i)
		}
	}
}

func TestPruneMessagesByTokensProtectsToolSequence(t *testing.T) {
	heavy := strings.Repeat("token ", 500)
	msgs := []api.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "old prompt " + heavy},
		{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{ID: "call_old", Name: "read_file", Arguments: `{"path":"go.mod"}`},
			},
		},
		{Role: "tool", ToolCallID: "call_old", Content: heavy},
		{Role: "assistant", Content: "old done"},
		{Role: "user", Content: "recent question"},
		{Role: "assistant", Content: "recent answer"},
	}

	pruned := pruneMessagesByTokens(msgs, 25, 300)
	if len(pruned) != 3 {
		t.Fatalf("expected 3 messages after token compaction, got %d: %+v", len(pruned), pruned)
	}
	if pruned[0].Role != "system" || pruned[1].Content != "recent question" {
		t.Errorf("unexpected pruned slice: %+v", pruned)
	}
}

func TestAgentLoopWithMockProvider(t *testing.T) {
	mock := provider.NewMock(
		api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{ID: "tc_1", Name: "read_file", Arguments: `{"path":"go.mod"}`},
			},
		},
		api.Message{
			Role:    "assistant",
			Content: "El módulo es agent-harness/Squib.",
		},
	)

	reg := testRegistry()

	gate := permission.NewGate(bufio.NewScanner(strings.NewReader("y\n")))

	initial := []api.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "¿Cuál es el nombre del módulo en go.mod?"},
	}

	finalMsgs := agentLoop(context.Background(), mock, reg, gate, initial)

	if len(mock.Calls) != 2 {
		t.Fatalf("expected MockProvider to be called 2 times, got %d", len(mock.Calls))
	}

	gotToolNames := make([]string, len(mock.Calls[0].Tools))
	for i, td := range mock.Calls[0].Tools {
		gotToolNames[i] = td.Name
	}
	wantOrder := "bash,copy_path,devops_status,find_files,list_directory,move_path,read_file,write_file"
	if strings.Join(gotToolNames, ",") != wantOrder {
		t.Errorf("tools passed to provider = %v, want %s", gotToolNames, wantOrder)
	}

	lastMsg := finalMsgs[len(finalMsgs)-1]
	if lastMsg.Role != "assistant" || !strings.Contains(lastMsg.Content, "agent-harness/Squib") {
		t.Errorf("unexpected final message: %+v", lastMsg)
	}
}

func TestVisibleContentStripsMultipleThinkBlocks(t *testing.T) {
	in := "<think>first thought</think>Hello <think>second thought</think>world<think>unclosed tail"
	if got, want := visibleContent(in), "Hello world"; got != want {
		t.Errorf("visibleContent() = %q, want %q", got, want)
	}

	orphan := "internal</think>  Final answer  "
	if got, want := visibleContent(orphan), "Final answer"; got != want {
		t.Errorf("visibleContent(orphan) = %q, want %q", got, want)
	}
}

func TestNeedsRevisionVerdict(t *testing.T) {
	cases := []struct {
		review string
		want   bool
	}{
		{"Todo correcto.\nAPPROVED", false},
		{"Falta un test.\nCHANGES_REQUESTED", true},
		{"No CHANGES_REQUESTED encontrados.\nAPPROVED", false},
		{"Primero APPROVED pero luego CHANGES_REQUESTED", true},
	}
	for _, tc := range cases {
		if got := needsRevision(tc.review); got != tc.want {
			t.Errorf("needsRevision(%q) = %v, want %v", tc.review, got, tc.want)
		}
	}
}

func testRolesMap() map[string]agents.Role {
	return map[string]agents.Role{
		"leader": {
			Name:         "leader",
			Model:        "openai/gpt-oss-20b",
			AllowedTools: map[string]bool{"read_file": true},
		},
		"implementer": {
			Name:         "implementer",
			Model:        "openai/gpt-oss-120b",
			AllowedTools: map[string]bool{"bash": true, "devops_status": true, "read_file": true, "write_file": true},
		},
		"reviewer": {
			Name:         "reviewer",
			Model:        "openai/gpt-oss-20b",
			AllowedTools: map[string]bool{"bash": true, "devops_status": true, "read_file": true},
		},
	}
}

func testRegistry() *tool.Registry {
	reg := tool.NewRegistry()
	reg.Register(tool.NewReadFileTool())
	reg.Register(tool.NewWriteFileTool())
	reg.Register(tool.NewBashTool())
	reg.Register(tool.NewDevOpsTool())
	reg.Register(tool.NewListDirectoryTool())
	reg.Register(tool.NewFindFilesTool())
	reg.Register(tool.NewCopyPathTool())
	reg.Register(tool.NewMovePathTool())
	return reg
}

func TestRunMultiAgentTurnHappyPathAndProgressFiles(t *testing.T) {
	roles := testRolesMap()
	reg := testRegistry()
	gate := permission.NewGate(bufio.NewScanner(strings.NewReader("")))
	progressDir := filepath.Join(t.TempDir(), "progress")

	var order []string
	leaderMock := provider.NewMock(
		api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{ID: "tc_blocked", Name: "write_file", Arguments: `{"path":"x.txt","content":"no"}`},
			},
		},
		api.Message{
			Role:    "assistant",
			Content: "<think>razonamiento oculto</think>Plan: actualizar main.go",
		},
	)
	implMock := provider.NewMock(api.Message{
		Role:    "assistant",
		Content: "<think>interno</think>Implementado cambio en main.go",
	})
	revMock := provider.NewMock(api.Message{
		Role:    "assistant",
		Content: "Todo verificado.\nAPPROVED",
	})

	newProvider := func(r agents.Role) provider.Provider {
		order = append(order, r.Name)
		switch r.Name {
		case "leader":
			return leaderMock
		case "implementer":
			return implMock
		default:
			return revMock
		}
	}

	summary, err := runMultiAgentTurn(context.Background(), systemPrompt, "arregla el bug", "", progressDir, roles, newProvider, reg, gate)
	if err != nil {
		t.Fatalf("runMultiAgentTurn returned error: %v", err)
	}
	if strings.Join(order, ",") != "leader,implementer,reviewer" {
		t.Errorf("execution order = %v, want leader,implementer,reviewer", order)
	}
	if !strings.Contains(summary, "APPROVED") {
		t.Errorf("expected summary to contain APPROVED, got %q", summary)
	}

	if len(leaderMock.Calls) != 2 {
		t.Fatalf("expected 2 leader calls, got %d", len(leaderMock.Calls))
	}
	lastLeaderMsg := leaderMock.Calls[1].Messages[len(leaderMock.Calls[1].Messages)-1]
	if !strings.Contains(lastLeaderMsg.Content, "not allowed") {
		t.Errorf("expected blocked tool message for leader, got %q", lastLeaderMsg.Content)
	}

	implPrompt := implMock.Calls[0].Messages[1].Content
	if strings.Contains(implPrompt, "razonamiento oculto") || !strings.Contains(implPrompt, "Plan: actualizar main.go") {
		t.Errorf("unexpected implementer prompt:\n%s", implPrompt)
	}

	for _, name := range []string{"leader", "implementer", "reviewer"} {
		data, err := os.ReadFile(filepath.Join(progressDir, name+".md"))
		if err != nil {
			t.Errorf("expected progress/%s.md to exist: %v", name, err)
		}
		if strings.Contains(string(data), "<think>") {
			t.Errorf("progress/%s.md leaked raw <think> tag", name)
		}
	}
}

func TestRunMultiAgentTurnStopsOnLeaderError(t *testing.T) {
	roles := testRolesMap()
	reg := testRegistry()
	gate := permission.NewGate(bufio.NewScanner(strings.NewReader("")))

	leaderMock := provider.NewMock()
	leaderMock.Err = errors.New("rate limit exceeded")
	implMock := provider.NewMock()
	revMock := provider.NewMock()

	var called []string
	newProvider := func(r agents.Role) provider.Provider {
		called = append(called, r.Name)
		switch r.Name {
		case "leader":
			return leaderMock
		case "implementer":
			return implMock
		default:
			return revMock
		}
	}

	_, err := runMultiAgentTurn(context.Background(), systemPrompt, "hola", "", t.TempDir(), roles, newProvider, reg, gate)
	if err == nil {
		t.Fatalf("expected error from leader to propagate")
	}
	if strings.Join(called, ",") != "leader" {
		t.Errorf("expected only leader to be invoked on error, got %v", called)
	}
}

func TestRunMultiAgentTurnRetriesOnceOnChangesRequested(t *testing.T) {
	roles := testRolesMap()
	reg := testRegistry()
	gate := permission.NewGate(bufio.NewScanner(strings.NewReader("")))

	var order []string
	leaderMock := provider.NewMock(api.Message{Role: "assistant", Content: "Plan inicial"})
	implCall := 0
	revCall := 0

	newProvider := func(r agents.Role) provider.Provider {
		order = append(order, r.Name)
		switch r.Name {
		case "leader":
			return leaderMock
		case "implementer":
			implCall++
			if implCall == 1 {
				return provider.NewMock(api.Message{Role: "assistant", Content: "Primer intento"})
			}
			return provider.NewMock(api.Message{Role: "assistant", Content: "Corrección aplicada"})
		default:
			revCall++
			if revCall == 1 {
				return provider.NewMock(api.Message{Role: "assistant", Content: "Falta validación.\nCHANGES_REQUESTED"})
			}
			return provider.NewMock(api.Message{Role: "assistant", Content: "Sigue faltando algo.\nCHANGES_REQUESTED"})
		}
	}

	_, err := runMultiAgentTurn(context.Background(), systemPrompt, "crea helper", "", t.TempDir(), roles, newProvider, reg, gate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "leader,implementer,reviewer,implementer,reviewer"
	if strings.Join(order, ",") != want {
		t.Errorf("revision order = %v, want %s", order, want)
	}
}

func TestLoadWorkspaceInstructionsAndBuildSystemPrompt(t *testing.T) {
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origCwd)
	})

	now := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)

	rulesPath, rulesContent := loadWorkspaceInstructions()
	if rulesPath != "" || rulesContent != "" {
		t.Fatalf("expected empty workspace instructions in empty dir, got (%q, %q)", rulesPath, rulesContent)
	}
	basePrompt := buildSystemPrompt(now, rulesPath, rulesContent)
	if !strings.Contains(basePrompt, "Current local date: 2026-11-26 (Thursday)") {
		t.Errorf("expected base prompt to contain current local date, got:\n%s", basePrompt)
	}

	reglasPath := filepath.Join(tmp, "06_Sistema", "Reglas-Squib.md")
	if err := os.MkdirAll(filepath.Dir(reglasPath), 0755); err != nil {
		t.Fatalf("failed to create 06_Sistema: %v", err)
	}
	if err := os.WriteFile(reglasPath, []byte("Bloc: 06_Sistema/Tareas pendientes.md"), 0644); err != nil {
		t.Fatalf("failed to write Reglas-Squib.md: %v", err)
	}

	rulesPath, rulesContent = loadWorkspaceInstructions()
	if rulesPath != "06_Sistema/Reglas-Squib.md" {
		t.Fatalf("loadWorkspaceInstructions() path = %q, want %q", rulesPath, "06_Sistema/Reglas-Squib.md")
	}
	vaultPrompt := buildSystemPrompt(now, rulesPath, rulesContent)
	if !strings.Contains(vaultPrompt, "auto-loaded from 06_Sistema/Reglas-Squib.md") ||
		!strings.Contains(vaultPrompt, "Bloc: 06_Sistema/Tareas pendientes.md") {
		t.Errorf("expected vault prompt to include rules path and content, got:\n%s", vaultPrompt)
	}

	squibMD := filepath.Join(tmp, "SQUIB.md")
	if err := os.WriteFile(squibMD, []byte("Custom SQUIB.md override"), 0644); err != nil {
		t.Fatalf("failed to write SQUIB.md: %v", err)
	}
	rulesPath, rulesContent = loadWorkspaceInstructions()
	if rulesPath != "SQUIB.md" || rulesContent != "Custom SQUIB.md override" {
		t.Fatalf("expected SQUIB.md to take priority, got (%q, %q)", rulesPath, rulesContent)
	}
}

func TestRunMultiAgentTurnDirectAnswerSkipsImplementerAndReviewer(t *testing.T) {
	roles := testRolesMap()
	reg := testRegistry()
	gate := permission.NewGate(bufio.NewScanner(strings.NewReader("")))
	progressDir := filepath.Join(t.TempDir(), "progress")

	if got := stripDirectAnswer("Tareas:\n- Revisar PR\n**DIRECT_ANSWER**"); got != "Tareas:\n- Revisar PR" {
		t.Fatalf("stripDirectAnswer did not remove wrapped marker: %q", got)
	}

	var called []string
	leaderMock := provider.NewMock(api.Message{
		Role:    "assistant",
		Content: "Tareas pendientes:\n- Terminar informe\n**DIRECT_ANSWER**",
	})
	newProvider := func(r agents.Role) provider.Provider {
		called = append(called, r.Name)
		if r.Name == "leader" {
			return leaderMock
		}
		t.Fatalf("unexpected invocation of role %q after DIRECT_ANSWER", r.Name)
		return nil
	}

	summary, err := runMultiAgentTurn(context.Background(), "SYS\n---\nReglas\n---", "Dime que tareas tengo pendientes", "", progressDir, roles, newProvider, reg, gate)
	if err != nil {
		t.Fatalf("runMultiAgentTurn failed: %v", err)
	}
	if strings.Join(called, ",") != "leader" {
		t.Errorf("called roles = %v, want only leader", called)
	}
	if !strings.Contains(summary, "Terminar informe") || strings.Contains(summary, "DIRECT_ANSWER") {
		t.Errorf("unexpected summary: %q", summary)
	}
	if !strings.Contains(leaderMock.Calls[0].Messages[0].Content, "Reglas") {
		t.Errorf("expected leader system prompt to include workspace rules")
	}

	data, err := os.ReadFile(filepath.Join(progressDir, "leader.md"))
	if err != nil {
		t.Fatalf("expected progress/leader.md to be written: %v", err)
	}
	if strings.Contains(string(data), "DIRECT_ANSWER") {
		t.Errorf("progress/leader.md must not contain DIRECT_ANSWER marker: %s", string(data))
	}
}

func TestRunMultiAgentTurnRecoversFromLeaderToolUseFailed(t *testing.T) {
	roles := testRolesMap()
	reg := testRegistry()
	gate := permission.NewGate(bufio.NewScanner(strings.NewReader("")))
	progressDir := filepath.Join(t.TempDir(), "progress")

	constraint := roleToolConstraint(roles["leader"])
	if !strings.Contains(constraint, "Allowed tools ONLY: read_file") || !strings.Contains(constraint, "NEVER call write_file") {
		t.Fatalf("unexpected roleToolConstraint: %q", constraint)
	}

	leaderMock := provider.NewMock()
	leaderCall := 0
	leaderMock.SendFunc = func(_ context.Context, _ []api.Message, tools []api.ToolDef) (*api.Message, error) {
		leaderCall++
		if leaderCall == 1 {
			return nil, &provider.ToolUseFailedError{
				StatusCode: 400,
				Message:    "Tool call validation failed: attempted to call tool 'write_file' which was not in request.tools",
			}
		}
		if len(tools) != 0 {
			t.Errorf("expected retry in text-only mode with 0 tools, got %d tools", len(tools))
		}
		return &api.Message{
			Role:    "assistant",
			Content: "Plan: añadir la nota en Ideas rápidas.md usando write_file.",
		}, nil
	}
	implMock := provider.NewMock(api.Message{Role: "assistant", Content: "Nota añadida en Ideas rápidas.md."})
	revMock := provider.NewMock(api.Message{Role: "assistant", Content: "Cambio verificado.\nAPPROVED"})

	newProvider := func(r agents.Role) provider.Provider {
		switch r.Name {
		case "leader":
			return leaderMock
		case "implementer":
			return implMock
		default:
			return revMock
		}
	}

	summary, err := runMultiAgentTurn(context.Background(), systemPrompt, "Añádeme una nota en ideas rápidas", "", progressDir, roles, newProvider, reg, gate)
	if err != nil {
		t.Fatalf("expected recovery to succeed, got error: %v", err)
	}
	if leaderCall != 2 {
		t.Errorf("leaderCall = %d, want 2", leaderCall)
	}
	if len(implMock.Calls) != 1 || len(revMock.Calls) != 1 {
		t.Errorf("expected implementer and reviewer to run after recovery (got impl=%d, rev=%d)", len(implMock.Calls), len(revMock.Calls))
	}
	if !strings.Contains(summary, "APPROVED") {
		t.Errorf("unexpected summary: %q", summary)
	}
}

func TestCompactBannerModels(t *testing.T) {
	roles := testRolesMap()
	if got := compactBannerModels(false, "", roles); got != "20b / 120b" {
		t.Errorf("compactBannerModels() = %q, want %q", got, "20b / 120b")
	}
	if got := compactBannerModels(true, "", roles); got != "mock" {
		t.Errorf("compactBannerModels(mock) = %q, want %q", got, "mock")
	}
	if got := compactBannerModels(false, "openai/gpt-oss-20b", roles); got != "20b" {
		t.Errorf("compactBannerModels(override) = %q, want %q", got, "20b")
	}
}
