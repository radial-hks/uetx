package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildCLI compiles the uetx binary once for the test binary's lifetime.
func buildCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "uetx")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build uetx: %v", err)
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// cmd/uetx → ../..
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runCLI(t *testing.T, bin string, stdin string, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repoRoot(t)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	return runResult{stdout.String(), stderr.String(), exit}
}

func TestCLI_UsageExitCode(t *testing.T) {
	bin := buildCLI(t)
	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{"no args", []string{}, 64},
		{"unknown command", []string{"foobar"}, 64},
		{"material without action", []string{"material"}, 64},
		{"version ok", []string{"version"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runCLI(t, bin, "", tc.args...)
			if got.exitCode != tc.wantExit {
				t.Fatalf("exit=%d want=%d stderr=%s", got.exitCode, tc.wantExit, got.stderr)
			}
		})
	}
}

func TestCLI_GenerateJSON(t *testing.T) {
	bin := buildCLI(t)
	got := runCLI(t, bin, "", "generate", "-i", "testdata/material/golden/M_WaterLevel.hlsl", "--seed", "42", "--json")
	if got.exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", got.exitCode, got.stderr)
	}
	var resp struct {
		OK           bool   `json:"ok"`
		T3D          string `json:"t3d"`
		MaterialName string `json:"materialName"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &resp); err != nil {
		t.Fatalf("parse JSON: %v\nout=%s", err, got.stdout)
	}
	if !resp.OK {
		t.Fatalf("ok=false: %s", got.stdout)
	}
	if !strings.Contains(resp.T3D, "Begin Object") {
		t.Fatalf("t3d missing 'Begin Object': %s", resp.T3D[:min(200, len(resp.T3D))])
	}
	if !strings.Contains(resp.T3D, "\r\n") {
		t.Fatalf("t3d missing CRLF")
	}
}

func TestCLI_InspectJSON(t *testing.T) {
	bin := buildCLI(t)
	got := runCLI(t, bin, "", "inspect", "-i", "testdata/material/golden/M_WaterLevel.hlsl", "--json")
	if got.exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", got.exitCode, got.stderr)
	}
	var resp struct {
		OK             bool `json:"ok"`
		InferredInputs []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"inferredInputs"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &resp); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if !resp.OK {
		t.Fatal("ok=false")
	}
	if len(resp.InferredInputs) == 0 {
		t.Fatal("no inputs inferred")
	}
	if resp.InferredInputs[0].Name != "WorldPosition" {
		t.Fatalf("first input name=%s want=WorldPosition", resp.InferredInputs[0].Name)
	}
}

func TestCLI_ValidateOK(t *testing.T) {
	bin := buildCLI(t)
	got := runCLI(t, bin, "", "validate", "-i", "testdata/material/golden/M_WaterLevel.hlsl", "--json")
	// exit 0 if no warnings, 2 if warnings-only. Both acceptable.
	if got.exitCode != 0 && got.exitCode != 2 {
		t.Fatalf("exit=%d (want 0 or 2) stderr=%s", got.exitCode, got.stderr)
	}
	var resp struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &resp); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if !resp.OK {
		t.Fatal("ok=false on a known-good fixture")
	}
}

func TestCLI_SeedReproducible(t *testing.T) {
	bin := buildCLI(t)
	a := runCLI(t, bin, "", "generate", "-i", "testdata/material/golden/M_WaterLevel.hlsl", "--seed", "12345", "--json")
	b := runCLI(t, bin, "", "generate", "-i", "testdata/material/golden/M_WaterLevel.hlsl", "--seed", "12345", "--json")
	if a.exitCode != 0 || b.exitCode != 0 {
		t.Fatalf("exits a=%d b=%d", a.exitCode, b.exitCode)
	}
	if a.stdout != b.stdout {
		t.Fatalf("seeded output not byte-identical")
	}

	// Different seed should yield different GUIDs (output differs).
	c := runCLI(t, bin, "", "generate", "-i", "testdata/material/golden/M_WaterLevel.hlsl", "--seed", "99999", "--json")
	if c.stdout == a.stdout {
		t.Fatalf("different seed produced identical output")
	}
}

func TestCLI_ConfigMergeCLIOverridesJSON(t *testing.T) {
	bin := buildCLI(t)
	cfg := `{"materialName":"FromConfig","outputType":"CMOT_Float4"}`
	cfgPath := filepath.Join(t.TempDir(), "cfg.json")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	// CLI -m FromCLI should win over config FromConfig.
	got := runCLI(t, bin, "",
		"generate",
		"-i", "testdata/material/golden/M_WaterLevel.hlsl",
		"-c", cfgPath,
		"-m", "FromCLI",
		"--seed", "1",
		"--json",
	)
	if got.exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", got.exitCode, got.stderr)
	}
	var resp struct {
		MaterialName string `json:"materialName"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.MaterialName != "FromCLI" {
		t.Fatalf("materialName=%q want=FromCLI (CLI must override JSON config)", resp.MaterialName)
	}

	// Config-only (no CLI -m) should yield FromConfig.
	got2 := runCLI(t, bin, "",
		"generate",
		"-i", "testdata/material/golden/M_WaterLevel.hlsl",
		"-c", cfgPath,
		"--seed", "1",
		"--json",
	)
	if got2.exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", got2.exitCode, got2.stderr)
	}
	var resp2 struct {
		MaterialName string `json:"materialName"`
	}
	if err := json.Unmarshal([]byte(got2.stdout), &resp2); err != nil {
		t.Fatal(err)
	}
	if resp2.MaterialName != "FromConfig" {
		t.Fatalf("materialName=%q want=FromConfig (config should apply when CLI flag absent)", resp2.MaterialName)
	}
}

func TestCLI_GenerateStdinJSON(t *testing.T) {
	bin := buildCLI(t)
	hlsl, err := os.ReadFile(filepath.Join(repoRoot(t), "testdata/material/golden/M_WaterLevel.hlsl"))
	if err != nil {
		t.Fatal(err)
	}
	req := map[string]any{
		"hlsl":         string(hlsl),
		"materialName": "M_FromStdin",
		"seed":         int64(7),
	}
	body, _ := json.Marshal(req)
	got := runCLI(t, bin, string(body), "generate", "--stdin-json", "--json")
	if got.exitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", got.exitCode, got.stderr)
	}
	var resp struct {
		OK           bool   `json:"ok"`
		MaterialName string `json:"materialName"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !resp.OK {
		t.Fatal("ok=false")
	}
	if resp.MaterialName != "M_FromStdin" {
		t.Fatalf("materialName=%q want=M_FromStdin", resp.MaterialName)
	}
}

func TestCLI_GenerateBusinessError(t *testing.T) {
	bin := buildCLI(t)
	// Malformed JSON via --stdin-json → E100 business error, exit 1.
	got := runCLI(t, bin, "{not json", "generate", "--stdin-json", "--json")
	if got.exitCode != 1 {
		t.Fatalf("exit=%d want=1 stderr=%s", got.exitCode, got.stderr)
	}
}

func TestParseInputSpec(t *testing.T) {
	tests := []struct {
		spec       string
		wantName   string
		wantType   string
		wantDef    string
		wantRGB    bool
		wantErr    bool
	}{
		{"foo:scalar", "foo", "scalar", "", false, false},
		{"foo:scalar:0.5", "foo", "scalar", "0.5", false, false},
		{"foo:vector:1,0,0,1:rgb", "foo", "vector", "1,0,0,1", true, false},
		{"foo:vector:1,0,0,1:true", "foo", "vector", "1,0,0,1", true, false},
		{"foo:vector:1,0,0,1:RGB", "foo", "vector", "1,0,0,1", true, false},
		{"foo:vector:1,0,0,1:false", "foo", "vector", "1,0,0,1", false, false},
		{"foo:vector:1,0,0,1:other", "foo", "vector", "1,0,0,1", false, false},
		{"bare", "", "", "", false, true},
	}
	for _, tc := range tests {
		t.Run(tc.spec, func(t *testing.T) {
			inp, err := parseInputSpec(tc.spec)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if inp.Name != tc.wantName || string(inp.Type) != tc.wantType ||
				inp.DefaultValue != tc.wantDef || inp.UseRGBMask != tc.wantRGB {
				t.Fatalf("got %+v", inp)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
