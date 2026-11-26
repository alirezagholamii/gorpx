package main

import "testing"

func TestInventory(t *testing.T) {
	paths := map[string]bool{"internal/static/example_test.go": true}
	source := []byte(`package static

import "testing"

func TestPath(t *testing.T) {
	t.Run("subtest", func(t *testing.T) {
		t.Skip("requires symlinks")
	})
}

func Testhelper() {}
func (fixture) TestMethod() {}
`)
	got, err := inventory(paths, func(string) ([]byte, error) { return source, nil })
	if err != nil {
		t.Fatal(err)
	}
	const owner = "internal/static/static.TestPath"
	if len(got.tests) != 1 || !got.tests[owner] {
		t.Fatalf("tests = %v, want %s", got.tests, owner)
	}
	want := skipCall{owner, `t.Skip("requires symlinks")`}
	if len(got.skips) != 1 || got.skips[want] != 1 {
		t.Fatalf("skips = %v, want %v", got.skips, want)
	}
}

func TestSecretPatterns(t *testing.T) {
	for _, tc := range []struct {
		line string
		want bool
	}{
		{line: "API_TOKEN" + `="sample-fixture-string-for-guard"`, want: true},
		{line: "token" + ` := "sample-fixture-string-for-guard"`, want: true},
		{line: "password" + ": replace_this_fixture_value", want: true},
		{line: "password" + ` = os.Getenv("PASSWORD")`},
		{line: "short token" + `="example"`},
	} {
		matched := false
		for _, pattern := range secretPatterns {
			matched = matched || pattern.MatchString(tc.line)
		}
		if matched != tc.want {
			t.Errorf("scan %q = %t, want %t", tc.line, matched, tc.want)
		}
	}
}
