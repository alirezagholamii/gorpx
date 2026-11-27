package evals

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCaseDefinitions(t *testing.T) {
	cases, err := fixtures.ReadDir("cases")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range cases {
		name, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok {
			continue
		}
		t.Run(name, func(t *testing.T) {
			loadCase(t, name)
			path := "cases/" + name + ".go.txt"
			data, err := fixtures.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parser.ParseFile(token.NewFileSet(), path, data, 0); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCheckTrace(t *testing.T) {
	session := agentEvent{Type: "session", SessionID: "session-1"}
	test := agentEvent{Type: "command", Command: "go test ./internal/server", Success: true, Output: "ok\tgithub.com/alirezagholamii/gorpx/internal/server\t0.005s\n"}
	complete := agentEvent{Type: "complete"}

	for _, tc := range []struct {
		name    string
		session string
		events  []agentEvent
		wantErr bool
	}{
		{name: "first turn", events: []agentEvent{session, test, complete}},
		{name: "resumed turn", session: session.SessionID, events: []agentEvent{test, complete}},
		{name: "missing session", events: []agentEvent{test, complete}, wantErr: true},
		{name: "different session", session: "another-session", events: []agentEvent{session, test, complete}, wantErr: true},
		{name: "invalid session", events: []agentEvent{{Type: "session", SessionID: "--last"}, test, complete}, wantErr: true},
		{name: "missing successful test", events: []agentEvent{session, {Type: "command", Command: test.Command, Output: test.Output}, complete}, wantErr: true},
		{name: "no test output", events: []agentEvent{session, {Type: "command", Command: test.Command, Success: true}, complete}, wantErr: true},
		{name: "incomplete turn", events: []agentEvent{session, test}, wantErr: true},
		{name: "failed turn", events: []agentEvent{session, test, complete, {Type: "failed"}}, wantErr: true},
		{name: "unexpected tool", events: []agentEvent{session, test, {Type: "external_tool", Tool: "remote"}, complete}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := newTurnCheck(tc.session, "go test")
			for _, event := range tc.events {
				check.observe(event)
			}
			got, err := check.result()
			if (err != nil) != tc.wantErr || (!tc.wantErr && got != session.SessionID) {
				t.Fatalf("result() = %q, %v, want error=%v", got, err, tc.wantErr)
			}
		})
	}
}

func TestRunnerTraces(t *testing.T) {
	const cmd = "go test ./internal/server"
	const output = "ok\\tgithub.com/alirezagholamii/gorpx/internal/server\\t0.005s\\n"

	for _, tc := range []struct {
		name, trace, resumeFlag, good, bad string
		args                               func(agentRequest) []string
		decode                             traceDecoder
	}{
		{
			name: "codex", resumeFlag: "resume", args: codexArgs, decode: decodeCodex,
			trace: `{"type":"thread.started","thread_id":"session-1"}
{"type":"item.completed","item":{"type":"command_execution","status":"completed","command":"/bin/bash -lc '` + cmd + `'","exit_code":0,"aggregated_output":"` + output + `"}}
{"type":"turn.completed"}`,
			good: `"exit_code":0`, bad: `"exit_code":1`,
		},
		{
			name: "claude", resumeFlag: "--resume", args: claudeArgs, decode: decodeClaude,
			trace: `{"type":"system","subtype":"init","session_id":"session-1"}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool-1","name":"Bash","input":{"command":"` + cmd + `"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool-1","is_error":false,"content":[{"type":"text","text":"` + output + `"}]}]}}
{"type":"result","subtype":"success","is_error":false,"session_id":"session-1"}`,
			good: `"is_error":false`, bad: `"is_error":true`,
		},
		{
			name: "gemini", resumeFlag: "--resume", args: geminiArgs, decode: decodeGemini,
			trace: `{"type":"init","session_id":"session-1"}
{"type":"tool_use","tool_id":"tool-1","tool_name":"run_shell_command","parameters":{"command":"` + cmd + `"}}
{"type":"tool_result","tool_id":"tool-1","status":"success","output":"` + output + `"}
{"type":"result","status":"success"}`,
			good: `"output":"` + output, bad: `"output":"` + output + `Exit Code: 1`,
		},
		{
			name: "opencode", resumeFlag: "--session", args: openCodeArgs, decode: decodeOpenCode,
			trace: `{"type":"step_start","sessionID":"session-1"}
{"type":"tool_use","sessionID":"session-1","part":{"tool":"bash","state":{"status":"completed","input":{"command":"` + cmd + `"},"output":"` + output + `","metadata":{"exit":0}}}}
{"type":"step_finish","sessionID":"session-1","part":{"reason":"tool-calls"}}
{"type":"step_start","sessionID":"session-1"}
{"type":"text","sessionID":"session-1","part":{"text":"Done"}}`,
			good: `"exit":0`, bad: `"exit":1`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const session = "session-1"
			check := newTurnCheck("", "go test")
			record := func(event agentEvent) error { check.observe(event); return nil }
			if err := tc.decode(strings.NewReader(tc.trace), record); err != nil {
				t.Fatal(err)
			}
			if got, err := check.result(); got != session || err != nil {
				t.Fatalf("result() = %q, %v", got, err)
			}
			if tc.name == "opencode" {
				nestedTrace := strings.Replace(tc.trace, `"metadata":{"exit":0}`, `"metadata":{"metadata":{"exitCode":0}}`, 1)
				check = newTurnCheck(session, "go test")
				if err := tc.decode(strings.NewReader(nestedTrace), record); err != nil {
					t.Fatal(err)
				}
				if _, err := check.result(); err != nil {
					t.Fatal(err)
				}
			}
			prompt := "follow-up -- with spaces"
			args := tc.args(agentRequest{Prompt: prompt, SessionID: session})
			resume := slices.Index(args, tc.resumeFlag)
			if resume < 0 || resume+1 >= len(args) || args[resume+1] != session || args[len(args)-1] != prompt {
				t.Fatalf("resume args = %q", args)
			}
			trace := strings.Replace(tc.trace, tc.good, tc.bad, 1)
			check = newTurnCheck(session, "go test")
			if err := tc.decode(strings.NewReader(trace), record); err != nil {
				t.Fatal(err)
			}
			if _, err := check.result(); err == nil {
				t.Fatal("failed shell command passed")
			}
		})
	}
}

func TestCustomRunner(t *testing.T) {
	dir := t.TempDir()
	wrapper := filepath.Join(t.TempDir(), "fixture-wrapper")
	const script = `#!/bin/sh
set -eu
test "$#" -eq 0
test "$GORPX_RUN_EVALS" = 0
test "$PWD" -ef .
cat > request.json
printf '%s\n' 'fixture stderr' >&2
printf '%s\n' '{"type":"session","session_id":"fixture:1"}' '{"type":"command","command":"go test ./internal/config","output":"ok github.com/alirezagholamii/gorpx/internal/config 0.005s","success":true}' '{"type":"complete"}'
`
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	runner, err := selectRunner("custom", wrapper)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GORPX_RUN_EVALS", "1")
	want := agentRequest{Prompt: "Follow-up with \"quotes\"\nand a new line", SessionID: "fixture:1"}
	check := newTurnCheck(want.SessionID, "go test")
	prefix := filepath.Join(t.TempDir(), "turn-02")
	if err := runner.Run(t.Context(), dir, prefix, want, func(event agentEvent) error {
		check.observe(event)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := check.result(); err != nil {
		t.Fatal(err)
	}
	request, err := os.ReadFile(filepath.Join(dir, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var received agentRequest
	if err := json.Unmarshal(request, &received); err != nil || received != want {
		t.Fatalf("request = %+v, %v, want %+v", received, err, want)
	}
	for _, suffix := range []string{".stdout.jsonl", ".stderr.log"} {
		info, err := os.Stat(prefix + suffix)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() == 0 || info.Mode().Perm() != 0600 {
			t.Fatalf("artifact %s has size %d and mode %v", suffix, info.Size(), info.Mode())
		}
	}
}

func TestSelectRunner(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"codex", "claude", "gemini", "opencode"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	for _, name := range []string{"", "claude", "gemini", "opencode"} {
		runner, err := selectRunner(name, "")
		if err != nil {
			t.Fatal(err)
		}
		want := name
		if want == "" {
			want = "codex"
		}
		if runner.Name() != want {
			t.Fatalf("runner = %q, want %q", runner.Name(), want)
		}
	}
	for _, name := range []string{"unknown", "custom"} {
		if _, err := selectRunner(name, ""); err == nil {
			t.Fatalf("expected selection error for %q", name)
		}
	}
}
