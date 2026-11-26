package evals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

func codexArgs(req agentRequest) []string {
	args := []string{"-a", "never", "exec", "--json", "--ignore-user-config", "--ignore-rules", "--sandbox", "workspace-write"}
	if req.SessionID != "" {
		args = append(args, "resume", req.SessionID)
	}
	return append(args, "--", req.Prompt)
}

func claudeArgs(req agentRequest) []string {
	args := []string{"--print", "--verbose", "--output-format", "stream-json",
		"--permission-mode", "acceptEdits", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
		"--allowedTools", "Bash(go test *)", "Bash(go fmt *)", "Bash(git status *)", "Bash(git diff *)"}
	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}
	return append(args, "--", req.Prompt)
}

func geminiArgs(req agentRequest) []string {
	args := []string{"--output-format", "stream-json", "--approval-mode", "auto_edit",
		"--allowed-tools", "ShellTool(go test)", "ShellTool(go fmt)"}
	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}
	return append(args, "--prompt", req.Prompt)
}

func openCodeArgs(req agentRequest) []string {
	args := []string{"run", "--format", "json"}
	if req.SessionID != "" {
		args = append(args, "--session", req.SessionID)
	}
	return append(args, "--", req.Prompt)
}

func decodeCodex(input io.Reader, emit func(agentEvent) error) error {
	type event struct {
		Type     string `json:"type"`
		ThreadID string `json:"thread_id"`
		Item     struct {
			Type     string `json:"type"`
			Command  string `json:"command"`
			Status   string `json:"status"`
			ExitCode *int   `json:"exit_code"`
			Output   string `json:"aggregated_output"`
		} `json:"item"`
	}
	return readEvents(input, func(e event) error {
		switch e.Type {
		case "thread.started":
			return emit(agentEvent{Type: "session", SessionID: e.ThreadID})
		case "turn.completed":
			return emit(agentEvent{Type: "complete"})
		case "turn.failed":
			return emit(agentEvent{Type: "failed"})
		case "item.started", "item.completed":
			item := e.Item
			if item.Type == "mcp_tool_call" {
				return emit(agentEvent{Type: "external_tool", Tool: item.Type})
			}
			if e.Type == "item.completed" && item.Type == "command_execution" {
				return emit(agentEvent{Type: "command", Command: item.Command, Output: item.Output,
					Success: item.Status == "completed" && item.ExitCode != nil && *item.ExitCode == 0})
			}
		}
		return nil
	})
}

func decodeClaude(input io.Reader, emit func(agentEvent) error) error {
	type block struct {
		Type      string          `json:"type"`
		ID        string          `json:"id"`
		Name      string          `json:"name"`
		ToolUseID string          `json:"tool_use_id"`
		Content   json.RawMessage `json:"content"`
		IsError   bool            `json:"is_error"`
		Input     struct {
			Command       string `json:"command"`
			RunBackground bool   `json:"run_in_background"`
		} `json:"input"`
	}
	type event struct {
		Type      string `json:"type"`
		Subtype   string `json:"subtype"`
		SessionID string `json:"session_id"`
		IsError   bool   `json:"is_error"`
		Message   struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	commands := make(map[string]string)
	return readEvents(input, func(e event) error {
		if (e.Type == "system" && e.Subtype == "init") || e.Type == "result" {
			if err := emit(agentEvent{Type: "session", SessionID: e.SessionID}); err != nil {
				return err
			}
		}
		switch e.Type {
		case "assistant", "user":
			content := bytes.TrimSpace(e.Message.Content)
			if !bytes.HasPrefix(content, []byte("[")) {
				return nil
			}
			var blocks []block
			if err := json.Unmarshal(content, &blocks); err != nil {
				return fmt.Errorf("decode Claude tool blocks: %w", err)
			}
			for _, block := range blocks {
				switch block.Type {
				case "tool_use":
					if strings.HasPrefix(block.Name, "mcp__") {
						if err := emit(agentEvent{Type: "external_tool", Tool: block.Name}); err != nil {
							return err
						}
					}
					if block.Name == "Bash" && !block.Input.RunBackground && block.ID != "" {
						commands[block.ID] = block.Input.Command
					}
				case "tool_result":
					if command, ok := commands[block.ToolUseID]; ok {
						delete(commands, block.ToolUseID)
						if err := emit(agentEvent{Type: "command", Command: command, Output: contentText(block.Content), Success: !block.IsError}); err != nil {
							return err
						}
					}
				}
			}
		case "result":
			return emit(completion(e.Subtype == "success" && !e.IsError))
		}
		return nil
	})
}

// Claude tool result content may be a string or an array of text blocks.
func contentText(data json.RawMessage) string {
	var text string
	if json.Unmarshal(data, &text) == nil {
		return text
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(data, &blocks) != nil {
		return ""
	}
	var lines []string
	for _, block := range blocks {
		if block.Type == "text" {
			lines = append(lines, block.Text)
		}
	}
	return strings.Join(lines, "\n")
}

var geminiExit = regexp.MustCompile(`(?m)^Exit Code:[ \t]*(-?\d+)[ \t]*\r?$`)

func decodeGemini(input io.Reader, emit func(agentEvent) error) error {
	type event struct {
		Type      string          `json:"type"`
		SessionID string          `json:"session_id"`
		ToolID    string          `json:"tool_id"`
		ToolName  string          `json:"tool_name"`
		Status    string          `json:"status"`
		Severity  string          `json:"severity"`
		Output    string          `json:"output"`
		Error     json.RawMessage `json:"error"`
		Params    struct {
			Command      string `json:"command"`
			IsBackground bool   `json:"is_background"`
		} `json:"parameters"`
	}
	commands := make(map[string]string)
	return readEvents(input, func(e event) error {
		switch e.Type {
		case "init":
			return emit(agentEvent{Type: "session", SessionID: e.SessionID})
		case "tool_use":
			switch e.ToolName {
			case "run_shell_command", "ShellTool":
				if e.ToolID != "" && !e.Params.IsBackground {
					commands[e.ToolID] = e.Params.Command
				}
			case "read_file", "write_file", "replace", "edit_file", "glob", "grep_search", "list_directory", "read_many_files", "write_todos", "activate_skill":
			default:
				return emit(agentEvent{Type: "external_tool", Tool: e.ToolName})
			}
		case "tool_result":
			if command, ok := commands[e.ToolID]; ok {
				delete(commands, e.ToolID)
				exit := geminiExit.FindStringSubmatch(e.Output)
				ok := e.Status == "success" && (len(e.Error) == 0 || string(e.Error) == "null") && (exit == nil || exit[1] == "0")
				return emit(agentEvent{Type: "command", Command: command, Output: strings.TrimPrefix(e.Output, "Output: "), Success: ok})
			}
		case "result":
			return emit(completion(e.Status == "success"))
		case "error":
			if e.Severity == "error" {
				return emit(agentEvent{Type: "failed"})
			}
		}
		return nil
	})
}

func decodeOpenCode(input io.Reader, emit func(agentEvent) error) error {
	type shellMetadata struct {
		Exit     *int `json:"exit"`
		ExitCode *int `json:"exitCode"`
	}
	type event struct {
		Type      string `json:"type"`
		SessionID string `json:"sessionID"`
		Part      struct {
			Tool   string `json:"tool"`
			Reason string `json:"reason"`
			State  struct {
				Status string `json:"status"`
				Output string `json:"output"`
				Input  struct {
					Command string `json:"command"`
				} `json:"input"`
				Metadata struct {
					shellMetadata
					Nested shellMetadata `json:"metadata"`
				} `json:"metadata"`
			} `json:"state"`
		} `json:"part"`
	}
	var session string
	var finished bool
	err := readEvents(input, func(e event) error {
		if e.SessionID != "" && e.SessionID != session {
			session = e.SessionID
			if err := emit(agentEvent{Type: "session", SessionID: session}); err != nil {
				return err
			}
		}
		switch e.Type {
		case "step_start":
			finished = false
		case "step_finish":
			switch e.Part.Reason {
			case "stop":
				finished = true
			case "tool-calls":
				finished = false
			case "error", "length", "content-filter":
				return emit(agentEvent{Type: "failed"})
			}
		case "text":
			// Some CLI versions exit after final text without emitting the last step_finish.
			finished = true
		case "error":
			return emit(agentEvent{Type: "failed"})
		case "tool_use":
			finished = false
			if e.Part.Tool == "bash" || e.Part.Tool == "shell" {
				state := e.Part.State
				// Older streams report exit status directly. Newer streams wrap tool metadata.
				var exit *int
				for _, candidate := range []*int{state.Metadata.Exit, state.Metadata.ExitCode, state.Metadata.Nested.Exit, state.Metadata.Nested.ExitCode} {
					if candidate != nil {
						exit = candidate
						break
					}
				}
				return emit(agentEvent{Type: "command", Command: state.Input.Command, Output: state.Output,
					Success: state.Status == "completed" && exit != nil && *exit == 0})
			}
			if !slices.Contains([]string{"read", "edit", "write", "apply_patch", "patch", "multiedit", "glob", "grep", "list", "todowrite", "todoread", "skill", "lsp"}, e.Part.Tool) {
				return emit(agentEvent{Type: "external_tool", Tool: e.Part.Tool})
			}
		}
		return nil
	})
	if err == nil && finished {
		return emit(agentEvent{Type: "complete"})
	}
	return err
}

func completion(success bool) agentEvent {
	if success {
		return agentEvent{Type: "complete"}
	}
	return agentEvent{Type: "failed"}
}
