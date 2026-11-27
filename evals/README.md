# Gorpx coding agent evaluations

Use Git, the Go version in `go.mod`, and an authenticated agent CLI on PATH.
Live runs make model calls and can send repository contents to the provider
you select. Run from the repository root with loopback networking available
for the Go integration tests.

```bash
GORPX_RUN_EVALS=1 AGENT_RUNNER=codex    go test -v -count=1 -timeout=30m ./evals
GORPX_RUN_EVALS=1 AGENT_RUNNER=claude   go test -v -count=1 -timeout=30m ./evals
GORPX_RUN_EVALS=1 AGENT_RUNNER=gemini   go test -v -count=1 -timeout=30m ./evals
GORPX_RUN_EVALS=1 AGENT_RUNNER=opencode go test -v -count=1 -timeout=30m ./evals
```

`AGENT_RUNNER` defaults to `codex`. To select one case, append
`-run 'TestAgentEvals/health-cache-revision$'` to the command.

| Runner | Headless entry point | Tool permissions |
| --- | --- | --- |
| `codex` | `codex exec --json` | Workspace-write sandbox and approvals disabled |
| `claude` | `claude --print --output-format stream-json` | Accepts edits and preapproves the fixture’s Go test commands |
| `gemini` | `gemini --approval-mode auto_edit --output-format stream-json` | Accepts edits and allows the fixture’s Go test commands |
| `opencode` | `opencode run --format json` | Uses local model and tool permissions. Allow worktree reads, edits, and the fixture’s Go test commands in your configured policy. |

Use CLI releases with the listed formats and session-resume support. See the
[Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference),
[Gemini CLI headless reference](https://geminicli.com/docs/cli/headless/), and
[OpenCode CLI reference](https://opencode.ai/docs/cli/commands).

The baseline defaults to committed `HEAD`. Commit the setup first so the
candidate sees the repository instructions and verifier. Set
`GORPX_EVAL_BASE` to a commit, tag, or branch to compare runs on a fixed
baseline. The tests expect the requested behavior to be absent at that
baseline. The harness runs generated Go code with the runner’s permissions.
Use a disposable runner for untrusted changes.

## Checks and artifacts

The harness creates detached worktrees in a temporary clone, reads each
agent’s reported session ID, and resumes the same session on follow-up turns.
Every turn must complete, record a successful Go test tool call, and stay
within the case’s file scope. The harness then builds the CLI, runs the race
suite, and runs acceptance assertions added after the agent exits. Ordinary
Go tests and `scripts/verify.sh` run the offline harness tests.

Each case writes `trace.jsonl` with normalized events tagged by runner and
turn, the raw `turn-NN.stdout.jsonl` and `turn-NN.stderr.log` files, and
`candidate.diff` after scope checks succeed. Results go under
`evals/results/` by default. Set `GORPX_EVAL_RESULTS` to change the location.
Review artifacts for prompts, source, and command output before sharing them.
Use the [rubric](RUBRIC.md) for a separate semantic review.

## Custom executable

```bash
GORPX_RUN_EVALS=1 AGENT_RUNNER=custom AGENT_RUNNER_COMMAND=/path/to/wrapper \
  go test -v -count=1 -timeout=30m ./evals
```

The executable is launched directly in the worktree with no arguments. It
receives one JSON object on stdin containing `prompt` and `session_id`. The
session ID is empty on the first turn. On follow-up turns, resume the supplied
session. Write one normalized JSON object per line to stdout. Use stderr for
diagnostics and return a nonzero exit status on process failure.

```jsonl
{"type":"session","session_id":"example-1"}
{"type":"command","command":"go test ./internal/config","output":"ok github.com/alirezagholamii/gorpx/internal/config 0.005s","success":true}
{"type":"complete"}
```

Emit `session` on the first turn and report the same ID if you emit it again.
Report actual shell results. Emit a `failed` event for a failed turn and an
`external_tool` event with the `tool` name for a nonlocal tool invocation.
Those events fail the checks.

To add a case, use a JSON file and matching `.go.txt` acceptance test in
`cases/`. Keep the file scope small and list follow-up requirements in
`turns`. Use a `TestHarness…` function to assert final observable behavior.
