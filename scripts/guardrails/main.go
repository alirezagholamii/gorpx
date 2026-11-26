// Command guardrails checks a candidate checkout against a trusted Git baseline.
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`-----BEGIN (?:[A-Z]+ )*PRIVATE KEY-----`),
	regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
	regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{30,}|xox[baprs]-[A-Za-z0-9-]{12,}|sk-(?:proj-)?[A-Za-z0-9_-]{20,})\b`),
	regexp.MustCompile(`(?i)\b(?:[A-Z0-9]+[_-])*(?:api[_-]?key|(?:(?:api|access|auth|secret)[_-]?)?token|password|(?:client[_-]?)?secret)["']?\s*(?::=|[:=])\s*["'][^"'\s]{16,}["']`),
	regexp.MustCompile(`(?im)^\s*(?:export\s+)?(?:[A-Z0-9]+[_-])*(?:api[_-]?key|token|password|client[_-]?secret)\s*[:=]\s*[A-Za-z0-9_+/.-]{20,}\s*(?:#.*)?$`),
	regexp.MustCompile(`(?i)\bBearer [A-Za-z0-9._~+/-]{24,}`),
}

func main() {
	repo := flag.String("repo", ".", "Git checkout containing the index")
	base := flag.String("base", "HEAD", "baseline commit")
	staged := flag.Bool("staged", false, "read candidate files from an exported index")
	flag.Parse()

	if err := verify(*repo, *base, *staged); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func verify(repo, ref string, staged bool) error {
	base, err := git(repo, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return err
	}
	revision := strings.TrimSpace(string(base))
	oldPaths, err := git(repo, "ls-tree", "-r", "--name-only", "-z", revision)
	if err != nil {
		return err
	}
	index := []string{"ls-files", "-z", "--cached"}
	if !staged {
		index = append(index, "--others", "--exclude-standard")
	}
	newPaths, err := git(repo, index...)
	if err != nil {
		return err
	}
	diff := []string{"diff", "--name-only", "-z", "--no-renames", "--diff-filter=ACMT"}
	if staged {
		diff = append(diff, "--cached")
	}
	changedPaths, err := git(repo, append(diff, revision, "--")...)
	if err != nil {
		return err
	}
	oldNames := names(oldPaths)
	newNames := names(newPaths)
	changed := names(changedPaths)

	// A path can also be new and untracked after a staged deletion.
	if !staged {
		untracked, err := git(repo, "ls-files", "--others", "--exclude-standard", "-z")
		if err != nil {
			return err
		}
		for path := range names(untracked) {
			changed[path] = true
		}
	}

	approved, err := exceptions()
	if err != nil {
		return err
	}
	var issues []string
	report := func(id, message string) {
		if !approved[id] {
			issues = append(issues, fmt.Sprintf("%s [%s]", message, id))
		}
	}

	for _, path := range []string{"go.mod", "go.sum"} {
		before, err := baselineFile(repo, revision, path, oldNames)
		if err != nil {
			return err
		}
		after, err := candidateFile(path)
		if err != nil {
			return err
		}
		if (before == nil) != (after == nil) || !bytes.Equal(before, after) {
			id := fingerprint("module", path, fileHash(before), fileHash(after))
			report(id, "unexpected dependency manifest change in "+path)
		}
	}

	oldTests, err := inventory(oldNames, func(path string) ([]byte, error) {
		return baselineFile(repo, revision, path, oldNames)
	})
	if err != nil {
		return err
	}
	newTests, err := inventory(newNames, candidateFile)
	if err != nil {
		return err
	}
	for name := range oldTests.tests {
		if !newTests.tests[name] {
			report(fingerprint("removed-test", name), "existing test missing: "+name)
		}
	}
	for skip, count := range newTests.skips {
		if count > oldTests.skips[skip] {
			id := fingerprint("new-skip", skip.owner, skip.call, fmt.Sprint(count))
			report(id, "new or changed skip call in "+skip.owner)
		}
	}

	for path := range changed {
		data, err := candidateFile(path)
		if err != nil {
			return err
		}
		if bytes.ContainsRune(data, 0) || !utf8.Valid(data) {
			continue
		}
		for line, content := range bytes.Split(data, []byte("\n")) {
			for _, pattern := range secretPatterns {
				if pattern.Match(content) {
					issues = append(issues, fmt.Sprintf("%s:%d: possible hardcoded credential", path, line+1))
					break
				}
			}
		}
	}

	for path := range newNames {
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		data, err := candidateFile(path)
		if err != nil {
			return err
		}
		if data == nil {
			continue
		}
		formatted, err := format.Source(data)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", path, err))
		} else if !bytes.Equal(data, formatted) {
			issues = append(issues, path+": run gofmt")
		}
	}

	if len(issues) > 0 {
		slices.Sort(issues)
		return errors.New(strings.Join(issues, "\n"))
	}
	fmt.Println("Repository guardrails and Go formatting passed.")
	return nil
}

type skipCall struct {
	owner string
	call  string
}

type testInventory struct {
	tests map[string]bool
	skips map[skipCall]int
}

func inventory(paths map[string]bool, read func(string) ([]byte, error)) (testInventory, error) {
	result := testInventory{tests: make(map[string]bool), skips: make(map[skipCall]int)}
	for path := range paths {
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := read(path)
		if err != nil {
			return result, err
		}
		if data == nil {
			continue
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, data, 0)
		if err != nil {
			return result, err
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			owner := filepath.ToSlash(filepath.Dir(path)) + "/" + file.Name.Name + "." + fn.Name.Name
			if fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
				suffix := strings.TrimPrefix(fn.Name.Name, "Test")
				first, _ := utf8.DecodeRuneInString(suffix)
				if !unicode.IsLower(first) {
					result.tests[owner] = true
				}
			}
			if fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				method, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				switch method.Sel.Name {
				case "Skip", "Skipf", "SkipNow":
					var text bytes.Buffer
					if err := format.Node(&text, token.NewFileSet(), call); err != nil {
						text.WriteString(method.Sel.Name)
					}
					result.skips[skipCall{owner, text.String()}]++
				}
				return true
			})
		}
	}
	return result, nil
}

func exceptions() (map[string]bool, error) {
	data, err := os.ReadFile("scripts/guardrails/exceptions.txt")
	if err != nil {
		return nil, err
	}
	approved := make(map[string]bool)
	for index, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		id, reason, ok := strings.Cut(line, " ")
		if !ok || strings.TrimSpace(reason) == "" || approved[id] {
			return nil, fmt.Errorf("invalid guardrail exception on line %d", index+1)
		}
		approved[id] = true
	}
	return approved, nil
}

func names(data []byte) map[string]bool {
	result := make(map[string]bool)
	for path := range bytes.SplitSeq(data, []byte{0}) {
		if len(path) > 0 {
			result[string(path)] = true
		}
	}
	return result
}

func candidateFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, nil
	}
	return os.ReadFile(path)
}

func baselineFile(repo, revision, path string, paths map[string]bool) ([]byte, error) {
	if !paths[path] {
		return nil, nil
	}
	data, err := git(repo, "cat-file", "blob", revision+":"+path)
	if err == nil && len(data) == 0 {
		return []byte{}, nil
	}
	return data, err
}

func fingerprint(kind string, values ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return fmt.Sprintf("%s:%x", kind, hash[:12])
}

func fileHash(data []byte) string {
	if data == nil {
		return "missing"
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func git(repo string, args ...string) ([]byte, error) {
	output, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, fmt.Errorf("git %s: %w: %s", args[0], err, exit.Stderr)
		}
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return output, nil
}
