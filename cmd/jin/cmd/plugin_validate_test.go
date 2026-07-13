package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takaaki-s/jind-ai/internal/exitcode"
)

// runValidateCmd invokes `plugin validate` with captured I/O. Flags are reset
// each call because cobra retains their last-set value between Executes and
// leaking, say, --fail-on-warning into a later test would give a phantom exit
// code.
func runValidateCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	_ = pluginValidateCmd.Flags().Set("manifest", "")
	_ = pluginValidateCmd.Flags().Set("registry", "")
	_ = pluginValidateCmd.Flags().Set("skip-uniqueness", "false")
	_ = pluginValidateCmd.Flags().Set("run-build", "false")
	_ = pluginValidateCmd.Flags().Set("fail-on-warning", "false")
	_ = pluginValidateCmd.Flags().Set("github-actions", "false")
	jsonOutput = false

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs(append([]string{"plugin", "validate"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	// pkg/plugin/manifest/testdata lives three parents up from this test's dir
	// (cmd/jin/cmd). Turning it into an absolute path so the fixture resolves
	// regardless of the CWD `go test` chose.
	base, err := filepath.Abs(filepath.Join("..", "..", "..", "pkg", "plugin", "manifest", "testdata", "manifests", name))
	if err != nil {
		t.Fatalf("abs fixture path: %v", err)
	}
	return base
}

func TestPluginValidateMinimalPasses(t *testing.T) {
	out, err := runValidateCmd(t, "--skip-uniqueness", fixturePath(t, "valid_minimal.yaml"))
	if err != nil {
		t.Fatalf("valid_minimal.yaml should pass: err=%v, out=%q", err, out)
	}
	if !strings.Contains(out, "0 ERROR") {
		t.Errorf("expected zero-error summary, got %q", out)
	}
}

func TestPluginValidateBadNameFails(t *testing.T) {
	out, err := runValidateCmd(t, "--skip-uniqueness", fixturePath(t, "invalid_bad_name.yaml"))
	if err == nil {
		t.Fatalf("bad name should fail, got out=%q", out)
	}
	var exitErr *exitcode.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != exitcode.GeneralError {
		t.Errorf("expected ExitError(GeneralError), got %v", err)
	}
	if !strings.Contains(out, "R5") {
		t.Errorf("expected rule R5 in output, got %q", out)
	}
}

func TestPluginValidateMissingManifestFailsWithRule1(t *testing.T) {
	out, err := runValidateCmd(t, "--skip-uniqueness", filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatalf("missing manifest should fail, got out=%q", out)
	}
	if !strings.Contains(out, "R1") {
		t.Errorf("expected rule R1 in output, got %q", out)
	}
}

func TestPluginValidateFailOnWarning(t *testing.T) {
	// valid_minimal.yaml has no adjacent LICENSE/README so PluginDir checks
	// emit two WARNs; without --fail-on-warning the run passes, with it we
	// expect exit 1.
	if _, err := runValidateCmd(t, "--skip-uniqueness", fixturePath(t, "valid_minimal.yaml")); err != nil {
		t.Fatalf("without --fail-on-warning WARN-only run must pass: %v", err)
	}
	out, err := runValidateCmd(t, "--skip-uniqueness", "--fail-on-warning", fixturePath(t, "valid_minimal.yaml"))
	if err == nil {
		t.Fatalf("--fail-on-warning should exit non-zero on WARN-only: out=%q", out)
	}
}

func TestPluginValidateGithubActionsAnnotations(t *testing.T) {
	out, err := runValidateCmd(t, "--skip-uniqueness", "--github-actions", fixturePath(t, "invalid_bad_name.yaml"))
	if err == nil {
		t.Fatalf("bad name should fail, out=%q", out)
	}
	if !strings.Contains(out, "::error file=invalid_bad_name.yaml") {
		t.Errorf("expected error annotation, got %q", out)
	}
	if !strings.Contains(out, "::warning file=invalid_bad_name.yaml") {
		t.Errorf("expected warning annotation, got %q", out)
	}
	if !strings.Contains(out, "title=R5") {
		t.Errorf("expected R5 title, got %q", out)
	}
}

func TestPluginValidateGithubStepSummary(t *testing.T) {
	summaryPath := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", summaryPath)
	if _, err := runValidateCmd(t, "--skip-uniqueness", "--github-actions", fixturePath(t, "invalid_bad_name.yaml")); err == nil {
		t.Fatal("bad name should fail")
	}
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "| Severity | Rule |") {
		t.Errorf("summary table header missing, got %q", body)
	}
	if !strings.Contains(body, "R5") {
		t.Errorf("summary missing R5 row, got %q", body)
	}
}

func TestPluginValidateRunBuildSuccess(t *testing.T) {
	// A minimal plugin dir: manifest whose build creates its entrypoint. Runs
	// bash so we only skip on hosts that don't have it.
	if _, err := os.Stat("/bin/bash"); err != nil {
		t.Skip("bash not available")
	}
	dir := t.TempDir()
	yaml := `schema_version: 1
name: build-ok
version: 0.1.0
description: run-build success fixture
jin: ">=0.0.0"
install:
  source:
    build:
      - "touch bin/hello"
      - "chmod +x bin/hello"
    entrypoint: ./bin/hello
`
	writeFile(t, filepath.Join(dir, "jind-ai-plugin.yaml"), yaml)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if _, err := runValidateCmd(t, "--skip-uniqueness", "--run-build", dir); err != nil {
		t.Fatalf("--run-build success case should pass: %v", err)
	}
}

func TestPluginValidateRunBuildFailure(t *testing.T) {
	if _, err := os.Stat("/bin/bash"); err != nil {
		t.Skip("bash not available")
	}
	dir := t.TempDir()
	yaml := `schema_version: 1
name: build-fail
version: 0.1.0
description: run-build failure fixture
jin: ">=0.0.0"
install:
  source:
    build:
      - "false"
    entrypoint: ./bin/hello
`
	writeFile(t, filepath.Join(dir, "jind-ai-plugin.yaml"), yaml)
	out, err := runValidateCmd(t, "--skip-uniqueness", "--run-build", dir)
	if err == nil {
		t.Fatalf("failing build should fail, out=%q", out)
	}
	if !strings.Contains(out, "R13") {
		t.Errorf("expected rule R13 in output, got %q", out)
	}
}

func TestPluginValidateRunBuildMissingEntrypoint(t *testing.T) {
	if _, err := os.Stat("/bin/bash"); err != nil {
		t.Skip("bash not available")
	}
	dir := t.TempDir()
	yaml := `schema_version: 1
name: build-noentry
version: 0.1.0
description: run-build entrypoint fixture
jin: ">=0.0.0"
install:
  source:
    build:
      - "true"
    entrypoint: ./bin/hello
`
	writeFile(t, filepath.Join(dir, "jind-ai-plugin.yaml"), yaml)
	out, err := runValidateCmd(t, "--skip-uniqueness", "--run-build", dir)
	if err == nil {
		t.Fatalf("missing entrypoint should fail, out=%q", out)
	}
	if !strings.Contains(out, "R14") {
		t.Errorf("expected rule R14 in output, got %q", out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
