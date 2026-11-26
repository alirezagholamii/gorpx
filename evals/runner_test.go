package evals

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type agentRequest struct {
	Prompt    string `json:"prompt"`
	SessionID string `json:"session_id"`
}

type agentEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Command   string `json:"command,omitempty"`
	Output    string `json:"output,omitempty"`
	Success   bool   `json:"success,omitempty"`
	Tool      string `json:"tool,omitempty"`
}

type AgentRunner interface {
	Name() string
	Run(ctx context.Context, worktree, artifactPrefix string, request agentRequest, emit func(agentEvent) error) error
}

type traceDecoder func(io.Reader, func(agentEvent) error) error

type cliRunner struct {
	name, executable string
	args             func(agentRequest) []string
	decode           traceDecoder
}

func (r cliRunner) Name() string { return r.name }

func selectRunner(name, customCommand string) (AgentRunner, error) {
	if name == "" {
		name = "codex"
	}
	runner := cliRunner{name: name}
	executable := name
	switch name {
	case "codex":
		runner.args, runner.decode = codexArgs, decodeCodex
	case "claude":
		runner.args, runner.decode = claudeArgs, decodeClaude
	case "gemini":
		runner.args, runner.decode = geminiArgs, decodeGemini
	case "opencode":
		runner.args, runner.decode = openCodeArgs, decodeOpenCode
	case "custom":
		if customCommand == "" {
			return nil, fmt.Errorf("set AGENT_RUNNER_COMMAND to a wrapper executable for AGENT_RUNNER=custom")
		}
		executable, runner.decode = customCommand, readEvents[agentEvent]
	default:
		return nil, fmt.Errorf("unknown AGENT_RUNNER %q (choose codex, claude, gemini, opencode or custom)", name)
	}
	path, err := exec.LookPath(executable)
	if err != nil {
		return nil, fmt.Errorf("find %s runner: %w", name, err)
	}
	runner.executable, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return runner, nil
}

func (r cliRunner) Run(ctx context.Context, dir, artifactPrefix string, req agentRequest, emit func(agentEvent) error) error {
	raw, err := os.OpenFile(artifactPrefix+".stdout.jsonl", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer raw.Close()
	stderr, err := os.OpenFile(artifactPrefix+".stderr.log", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer stderr.Close()

	var args []string
	if r.args != nil {
		args = r.args(req)
	}
	cmd := command(ctx, dir, r.executable, args...)
	cmd.Stdout, cmd.Stderr = raw, stderr
	if r.args == nil {
		body, err := json.Marshal(req)
		if err != nil {
			return err
		}
		cmd.Stdin = bytes.NewReader(append(body, '\n'))
	}
	runErr := cmd.Run()
	if ctx.Err() != nil {
		runErr = errors.Join(runErr, ctx.Err())
	}
	if _, err := raw.Seek(0, io.SeekStart); err != nil {
		return errors.Join(runErr, err)
	}
	return errors.Join(runErr, r.decode(raw, emit))
}

func readEvents[T any](input io.Reader, handle func(T) error) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		var event T
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("decode JSONL event on line %d: %w", line, err)
		}
		if err := handle(event); err != nil {
			return err
		}
	}
	return scanner.Err()
}
