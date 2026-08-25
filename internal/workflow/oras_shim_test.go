package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// installFakeOras puts a fake `oras` executable first on PATH for the duration
// of the test. The fake appends its working directory and arguments to a log
// file and prints stdout (if non-empty) before exiting with exitCode.
// It returns a function that reads back the recorded invocations.
func installFakeOras(t *testing.T, stdout string, exitCode int) func() []string {
	t.Helper()
	return installFakeOrasSequence(t, []string{stdout}, exitCode)
}

// installFakeOrasSequence is like installFakeOras but answers the n-th
// invocation with responses[n]. Invocations beyond the slice reuse the last
// response, so a single-element slice answers every call the same way.
func installFakeOrasSequence(t *testing.T, responses []string, exitCode int) func() []string {
	t.Helper()
	dir := t.TempDir()
	logFile := filepath.Join(dir, "calls.log")
	counterFile := filepath.Join(dir, "counter")
	for i, r := range responses {
		if err := os.WriteFile(filepath.Join(dir, "stdout."+itoa(i)), []byte(r), 0644); err != nil {
			t.Fatal(err)
		}
	}
	last := itoa(len(responses) - 1)
	script := "#!/bin/sh\n" +
		"printf '%s\\t%s\\n' \"$(pwd)\" \"$*\" >> \"" + logFile + "\"\n" +
		"n=$(cat \"" + counterFile + "\" 2>/dev/null || echo 0)\n" +
		"echo $((n+1)) > \"" + counterFile + "\"\n" +
		"f=\"" + dir + "/stdout.$n\"\n" +
		"[ -f \"$f\" ] || f=\"" + dir + "/stdout." + last + "\"\n" +
		"cat \"$f\"\n" +
		"exit " + itoa(exitCode) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "oras"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return func() []string {
		data, err := os.ReadFile(logFile)
		if err != nil {
			return nil
		}
		return strings.Split(strings.TrimSpace(string(data)), "\n")
	}
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
