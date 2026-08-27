package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTool is a fake executable installed first on PATH by installFakeTool.
type fakeTool struct {
	dir string
}

// fakeToolResponse is what a fake tool prints and exits with for one call.
type fakeToolResponse struct {
	stdout   string
	exitCode int
}

// installFakeTool puts a fake executable called name (e.g. "oras", "modctl")
// first on PATH for the duration of the test. The fake appends its working
// directory and arguments to a log file, keeps a copy of every argument that
// names a regular file (callers often delete their temp files right after
// the call), prints the n-th entry of responses on its n-th invocation and
// exits with exitCode. Invocations beyond the slice reuse the last response,
// so a single-element slice answers every call the same way.
func installFakeTool(t *testing.T, name string, responses []string, exitCode int) *fakeTool {
	t.Helper()
	fakes := make([]fakeToolResponse, len(responses))
	for i, r := range responses {
		fakes[i] = fakeToolResponse{stdout: r, exitCode: exitCode}
	}
	return installFakeToolResponses(t, name, fakes)
}

// installFakeToolResponses is like installFakeTool but answers the n-th
// invocation with responses[n], each with its own stdout and exit code.
func installFakeToolResponses(t *testing.T, name string, responses []fakeToolResponse) *fakeTool {
	t.Helper()
	dir := t.TempDir()
	logFile := filepath.Join(dir, "calls.log")
	counterFile := filepath.Join(dir, "counter")
	for i, r := range responses {
		if err := os.WriteFile(filepath.Join(dir, "stdout."+itoa(i)), []byte(r.stdout), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "exit."+itoa(i)), []byte(itoa(r.exitCode)), 0644); err != nil {
			t.Fatal(err)
		}
	}
	last := itoa(len(responses) - 1)
	script := "#!/bin/sh\n" +
		"printf '%s\\t%s\\n' \"$(pwd)\" \"$*\" >> \"" + logFile + "\"\n" +
		"n=$(cat \"" + counterFile + "\" 2>/dev/null || echo 0)\n" +
		"echo $((n+1)) > \"" + counterFile + "\"\n" +
		"for a in \"$@\"; do [ -f \"$a\" ] && cp \"$a\" \"" + dir + "/arg.$n.$(basename \"$a\")\"; done\n" +
		"f=\"" + dir + "/stdout.$n\"\n" +
		"[ -f \"$f\" ] || f=\"" + dir + "/stdout." + last + "\"\n" +
		"e=\"" + dir + "/exit.$n\"\n" +
		"[ -f \"$e\" ] || e=\"" + dir + "/exit." + last + "\"\n" +
		"cat \"$f\"\n" +
		"exit $(cat \"$e\")\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &fakeTool{dir: dir}
}

// calls returns the recorded invocations, one "pwd\targs..." line each.
func (f *fakeTool) calls() []string {
	data, err := os.ReadFile(filepath.Join(f.dir, "calls.log"))
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// file returns the copy of the file named base that the call-th invocation
// (0-based) received as an argument.
func (f *fakeTool) file(t *testing.T, call int, base string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.dir, "arg."+itoa(call)+"."+base))
	if err != nil {
		t.Fatalf("call %d did not receive a file named %s: %v", call, base, err)
	}
	return data
}

// installFakeOras installs a fake `oras` answering every call with stdout
// and returns a function that reads back the recorded invocations.
func installFakeOras(t *testing.T, stdout string, exitCode int) func() []string {
	t.Helper()
	return installFakeTool(t, "oras", []string{stdout}, exitCode).calls
}

// installFakeOrasSequence is like installFakeOras but answers the n-th
// invocation with responses[n].
func installFakeOrasSequence(t *testing.T, responses []string, exitCode int) func() []string {
	t.Helper()
	return installFakeTool(t, "oras", responses, exitCode).calls
}

// fakeOrasResponse is what the fake oras prints and exits with for one call.
type fakeOrasResponse struct {
	stdout   string
	exitCode int
}

// installFakeOrasResponses answers the n-th oras invocation with
// responses[n], each with its own stdout and exit code. Invocations beyond
// the slice reuse the last response.
func installFakeOrasResponses(t *testing.T, responses []fakeOrasResponse) func() []string {
	t.Helper()
	if len(responses) == 0 {
		t.Fatal("installFakeOrasResponses needs at least one response")
	}
	fakes := make([]fakeToolResponse, len(responses))
	for i, r := range responses {
		fakes[i] = fakeToolResponse(r)
	}
	return installFakeToolResponses(t, "oras", fakes).calls
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
