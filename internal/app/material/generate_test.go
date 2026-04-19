package material

import (
	"os"
	"strings"
	"testing"

	"github.com/radial/uetx/internal/domain"
)

func TestGenerate_WaterLevel(t *testing.T) {
	hlsl, err := os.ReadFile("../../../testdata/material/golden/M_WaterLevel.hlsl")
	if err != nil {
		t.Fatalf("read HLSL: %v", err)
	}

	resp := Generate(domain.GenerateRequest{
		HLSL:         string(hlsl),
		MaterialName: "M_WaterLevel",
		Seed:         42,
	})

	if !resp.OK {
		t.Fatalf("generate failed: %v", resp.Errors)
	}

	if resp.T3D == "" {
		t.Fatal("T3D output is empty")
	}

	if resp.EffectiveOutputType != domain.CMOTFloat4 {
		t.Errorf("outputType = %q, want CMOT_Float4", resp.EffectiveOutputType)
	}

	if len(resp.InferredInputs) != 7 {
		t.Errorf("inferred inputs = %d, want 7", len(resp.InferredInputs))
	}

	if resp.Stats == nil {
		t.Fatal("stats is nil")
	}
	if resp.Stats.NodeCount != 10 {
		t.Errorf("nodeCount = %d, want 10", resp.Stats.NodeCount)
	}
	if !resp.Stats.HasBreakOut {
		t.Error("expected hasBreakOut = true")
	}

	// CRLF check
	for i, c := range resp.T3D {
		if c == '\n' && (i == 0 || resp.T3D[i-1] != '\r') {
			t.Fatal("found bare LF in T3D output")
		}
	}
}

func TestGenerate_WithExplicitInputs(t *testing.T) {
	resp := Generate(domain.GenerateRequest{
		HLSL: "/* empty template */\nreturn float3(1,0,0);",
		Inputs: []domain.NodeInput{
			{Name: "Speed", Type: domain.ParamScalar, DefaultValue: "2.0"},
		},
		OutputType: domain.CMOTFloat3,
		Seed:       99,
	})

	if !resp.OK {
		t.Fatalf("generate failed: %v", resp.Errors)
	}
	if !strings.Contains(resp.T3D, "MaterialExpressionScalarParameter") {
		t.Error("expected scalar parameter in output")
	}
}

func TestGenerate_InvalidMaterialName(t *testing.T) {
	resp := Generate(domain.GenerateRequest{
		HLSL:         "/* Pin 0 Name: [X] | Type suggestion: Scalar */\nreturn X;",
		MaterialName: "M Bad Name!",
		Seed:         1,
	})

	if resp.OK {
		t.Fatal("expected failure for invalid material name")
	}
	hasE110 := false
	for _, e := range resp.Errors {
		if e.Code == "E110" {
			hasE110 = true
		}
	}
	if !hasE110 {
		t.Error("expected E110 error")
	}
}

func TestGenerate_EmptyHLSL(t *testing.T) {
	resp := Generate(domain.GenerateRequest{HLSL: ""})
	if resp.OK {
		t.Fatal("expected failure for empty HLSL")
	}
}

func TestGenerate_Idempotent(t *testing.T) {
	hlsl := "/* Pin 0 Name: [A] | Type suggestion: Scalar */\nreturn A;"
	r1 := Generate(domain.GenerateRequest{HLSL: hlsl, Seed: 123})
	r2 := Generate(domain.GenerateRequest{HLSL: hlsl, Seed: 123})
	if r1.T3D != r2.T3D {
		t.Fatal("same seed did not produce identical T3D")
	}
}

func TestInspect(t *testing.T) {
	hlsl, err := os.ReadFile("../../../testdata/material/golden/M_WaterLevel.hlsl")
	if err != nil {
		t.Fatalf("read HLSL: %v", err)
	}
	resp := Inspect(string(hlsl))
	if !resp.OK {
		t.Fatalf("inspect failed: %v", resp.Errors)
	}
	if len(resp.InferredInputs) != 7 {
		t.Errorf("inferred inputs = %d, want 7", len(resp.InferredInputs))
	}
	if resp.T3D != "" {
		t.Error("inspect should not produce T3D")
	}
}
