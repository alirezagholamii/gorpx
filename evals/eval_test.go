package evals

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

//go:embed cases/*.json cases/*.go.txt
var fixtures embed.FS

type evalCase struct {
	Turns           []string `json:"turns"`
	AllowedFiles    []string `json:"allowed_files"`
	RequiredFile    string   `json:"required_file"`
	CheckPackage    string   `json:"check_package"`
	RequiredCommand string   `json:"required_command"`
}

func TestAgentEvals(t *testing.T) {
	if os.Getenv("GORPX_RUN_EVALS") != "1" {
		t.Skip("set GORPX_RUN_EVALS=1 to run live agent evaluations")
	}
	runner, err := selectRunner(os.Getenv("AGENT_RUNNER"), os.Getenv("AGENT_RUNNER_COMMAND"))
	if err != nil {
		t.Fatal(err)
	}
	root := strings.TrimSpace(string(run(t, ".", "git", "rev-parse", "--show-toplevel")))
	ref := os.Getenv("GORPX_EVAL_BASE")
	if ref == "" {
		ref = "HEAD"
	}
	base := strings.TrimSpace(string(run(t, root, "git", "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")))

	results := os.Getenv("GORPX_EVAL_RESULTS")
	if results == "" {
		results = filepath.Join(root, "evals", "results")
	}
	results, err = filepath.Abs(results)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(results, 0700); err != nil {
		t.Fatal(err)
	}
	results, err = os.MkdirTemp(results, "run-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("runner %s; baseline %s; artifacts %s", runner.Name(), base, results)

	// Register worktrees against a temporary clone so the source Git metadata
	// and working directory are independent of what the agent does.
	scratch := t.TempDir()
	anchor := filepath.Join(scratch, "anchor")
	run(t, scratch, "git", "clone", "--no-checkout", "--no-hardlinks", "--", root, anchor)

	cases, err := fixtures.ReadDir("cases")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range cases {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			spec := loadCase(t, name)
			dir := filepath.Join(scratch, name)
			run(t, anchor, "git", "worktree", "add", "--detach", dir, base)
			// The parent temp directory still cleans up if the agent damages the worktree.
			t.Cleanup(func() {
				run(t, anchor, "git", "worktree", "remove", "--force", dir)
			})
			runCase(t, runner, dir, base, filepath.Join(results, name), name, spec)
		})
	}
}

func loadCase(t *testing.T, name string) evalCase {
	t.Helper()
	data, err := fixtures.ReadFile("cases/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var spec evalCase
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		t.Fatal(err)
	}
	if len(spec.Turns) == 0 || spec.RequiredFile == "" || !slices.Contains(spec.AllowedFiles, spec.RequiredFile) ||
		!strings.HasPrefix(spec.CheckPackage, "internal/") || !filepath.IsLocal(spec.CheckPackage) ||
		spec.RequiredCommand == "" {
		t.Fatalf("incomplete case %s", name)
	}
	for _, turn := range spec.Turns {
		if strings.TrimSpace(turn) == "" {
			t.Fatalf("case %s has an empty turn", name)
		}
	}
	return spec
}

func runCase(t *testing.T, runner AgentRunner, dir, base, artifacts, name string, spec evalCase) {
	t.Helper()
	if err := os.Mkdir(artifacts, 0700); err != nil {
		t.Fatal(err)
	}
	trace, err := os.OpenFile(filepath.Join(artifacts, "trace.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer trace.Close()

	encoder := json.NewEncoder(trace)
	var session string
	for i, prompt := range spec.Turns {
		turn := i + 1
		if i == 0 {
			prompt = "Read the repository's AGENTS.md if present and follow it within the task's file scope.\n\n" + prompt
		}
		check := newTurnCheck(session, spec.RequiredCommand)
		record := func(event agentEvent) error {
			entry := struct {
				Runner string `json:"runner"`
				Turn   int    `json:"turn"`
				agentEvent
			}{runner.Name(), turn, event}
			if err := encoder.Encode(entry); err != nil {
				return err
			}
			check.observe(event)
			return nil
		}
		prefix := filepath.Join(artifacts, fmt.Sprintf("turn-%02d", turn))
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		err := runner.Run(ctx, dir, prefix, agentRequest{Prompt: prompt, SessionID: session}, record)
		cancel()
		if err != nil {
			t.Fatalf("%s turn %d: %v (artifacts in %s)", runner.Name(), turn, err, artifacts)
		}
		session, err = check.result()
		if err != nil {
			t.Fatalf("%s turn %d: %v (trace in %s)", runner.Name(), turn, err, artifacts)
		}
		assertScope(t, dir, base, spec)
	}

	run(t, dir, "git", "add", "-N", "--", ".")
	patch := run(t, dir, "git", "diff", "--binary", "--no-ext-diff", "--no-textconv", base, "--")
	if err := os.WriteFile(filepath.Join(artifacts, "candidate.diff"), patch, 0600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "go", "build", "-mod=readonly", "-o", filepath.Join(t.TempDir(), "gorpx"), "./cmd/gorpx")
	run(t, dir, "go", "test", "-mod=readonly", "-race", "-count=1", "./...")

	check, err := fixtures.ReadFile("cases/" + name + ".go.txt")
	if err != nil {
		t.Fatal(err)
	}
	checkPath := filepath.Join(dir, spec.CheckPackage, "gorpx_harness_acceptance_test.go")
	file, err := os.OpenFile(checkPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.Write(check)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("write acceptance tests: write=%v close=%v", writeErr, closeErr)
	}
	output := run(t, dir, "go", "test", "-mod=readonly", "-race", "-count=1", "-v", "-run", "^TestHarness", "./"+spec.CheckPackage)
	if !bytes.Contains(output, []byte("--- PASS: TestHarness")) ||
		bytes.Contains(output, []byte("--- SKIP: TestHarness")) {
		t.Fatalf("acceptance test did not pass:\n%s", output)
	}
}

func assertScope(t *testing.T, dir, base string, spec evalCase) {
	t.Helper()
	if head := strings.TrimSpace(string(run(t, dir, "git", "rev-parse", "HEAD"))); head != base {
		t.Fatalf("agent moved HEAD to %s", head)
	}

	// Check both the index and the final tree. Include ignored untracked files.
	index := run(t, dir, "git", "diff", "--cached", "--name-only", "--no-renames", "-z", base, "--")
	tree := run(t, dir, "git", "diff", "--name-only", "--no-renames", "-z", base, "--")
	others := run(t, dir, "git", "ls-files", "--others", "-z")
	for _, paths := range [][]byte{index, tree, others} {
		for path := range bytes.SplitSeq(paths, []byte{0}) {
			if len(path) != 0 && !slices.Contains(spec.AllowedFiles, string(path)) {
				t.Errorf("out-of-scope change: %q", path)
			}
		}
	}
	if !slices.ContainsFunc(bytes.Split(tree, []byte{0}), func(path []byte) bool {
		return string(path) == spec.RequiredFile
	}) {
		t.Errorf("required implementation file %q was not changed", spec.RequiredFile)
	}
	if t.Failed() {
		t.FailNow()
	}
}

var (
	sessionID   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,127}$`)
	testFailure = regexp.MustCompile(`(?m)^(?:FAIL(?:[ \t]|$)|--- FAIL:)`)
	testResult  = regexp.MustCompile(`(?m)^ok[ \t]+github\.com/alirezagholamii/gorpx(?:/|[ \t])`)
)

type turnCheck struct {
	session, requiredCommand string
	commandPattern           *regexp.Regexp
	completed, commandPassed bool
	err                      error
}

func newTurnCheck(session, requiredCommand string) *turnCheck {
	return &turnCheck{
		session: session, requiredCommand: requiredCommand,
		commandPattern: regexp.MustCompile(`\b` + regexp.QuoteMeta(requiredCommand) + `\b`),
	}
}

func (c *turnCheck) observe(event agentEvent) {
	if c.err != nil {
		return
	}
	switch event.Type {
	case "session":
		if !sessionID.MatchString(event.SessionID) || (c.session != "" && c.session != event.SessionID) {
			c.err = fmt.Errorf("unexpected session ID %q", event.SessionID)
			return
		}
		c.session = event.SessionID
	case "command":
		if event.Success && c.commandPattern.MatchString(event.Command) && testResult.MatchString(event.Output) && !testFailure.MatchString(event.Output) {
			c.commandPassed = true
		}
	case "complete":
		c.completed = true
	case "failed":
		c.err = fmt.Errorf("agent reported a failed turn")
	case "external_tool":
		c.err = fmt.Errorf("unexpected external tool %q", event.Tool)
	default:
		c.err = fmt.Errorf("unknown normalized event %q", event.Type)
	}
}

func (c *turnCheck) result() (string, error) {
	if c.err != nil {
		return "", c.err
	}
	if c.session == "" || !c.completed || !c.commandPassed {
		return "", fmt.Errorf("missing session, completed turn, or successful %q tool call", c.requiredCommand)
	}
	return c.session, nil
}

func command(ctx context.Context, dir, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GORPX_RUN_EVALS=0", "GOWORK=off", "GOFLAGS=-mod=readonly")
	cmd.WaitDelay = 5 * time.Second
	return cmd
}

func run(t *testing.T, dir, name string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var stderr bytes.Buffer
	cmd := command(ctx, dir, name, args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s%s", name, strings.Join(args, " "), err, out, stderr.String())
	}
	return out
}
