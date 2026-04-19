package parser

import (
	"os"
	"testing"

	"github.com/radial/uetx/internal/domain"
)

func TestInferParamType(t *testing.T) {
	tests := []struct {
		input string
		want  domain.ParamType
	}{
		{"Vector 3", domain.ParamVector},
		{"Scalar", domain.ParamScalar},
		{"Scalar (推荐: 暴露为材质参数)", domain.ParamScalar},
		{"Vector 3 (水上基础颜色 RGB)", domain.ParamVector},
		{"World Position", domain.ParamWorldPosition},
		{"Absolute World Position", domain.ParamWorldPosition},
		{"Time", domain.ParamTime},
		{"TextureCoordinate", domain.ParamUV},
		{"Texture Coordinate", domain.ParamUV},
		{"UV", domain.ParamUV},
		{"Color", domain.ParamVector},
		{"float", domain.ParamScalar},
	}
	for _, tt := range tests {
		got := InferParamType(tt.input)
		if got != tt.want {
			t.Errorf("InferParamType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseTemplateWaterLevel(t *testing.T) {
	hlsl, err := os.ReadFile("../../../testdata/material/golden/M_WaterLevel.hlsl")
	if err != nil {
		t.Fatalf("read HLSL: %v", err)
	}

	result, diags := ParseTemplate(string(hlsl))

	for _, d := range diags {
		if d.Code[0] == 'E' {
			t.Fatalf("unexpected error diagnostic: %s: %s", d.Code, d.Message)
		}
	}

	if len(result.Inputs) != 7 {
		t.Fatalf("got %d inputs, want 7", len(result.Inputs))
	}

	wantInputs := []struct {
		name string
		typ  domain.ParamType
	}{
		{"WorldPosition", domain.ParamVector},
		{"WaterLevel", domain.ParamScalar},
		{"Feather", domain.ParamScalar},
		{"AboveWaterColor", domain.ParamVector},
		{"AboveWaterAlpha", domain.ParamScalar},
		{"UnderwaterColor", domain.ParamVector},
		{"UnderwaterAlpha", domain.ParamScalar},
	}
	for i, w := range wantInputs {
		if result.Inputs[i].Name != w.name {
			t.Errorf("input[%d].Name = %q, want %q", i, result.Inputs[i].Name, w.name)
		}
		if result.Inputs[i].Type != w.typ {
			t.Errorf("input[%d].Type = %q, want %q", i, result.Inputs[i].Type, w.typ)
		}
	}

	if !result.HasOutputType {
		t.Fatal("output type not found")
	}
	if result.OutputType != domain.CMOTFloat4 {
		t.Errorf("outputType = %q, want CMOT_Float4", result.OutputType)
	}
}

func TestParseTemplateEmptyHLSL(t *testing.T) {
	_, diags := ParseTemplate("")
	if len(diags) == 0 || diags[0].Code != "E001" {
		t.Fatalf("expected E001 for empty HLSL, got %v", diags)
	}
}

func TestParseTemplateNoCommentBlock(t *testing.T) {
	_, diags := ParseTemplate("float x = 1.0;\nreturn x;")
	if len(diags) == 0 || diags[0].Code != "E002" {
		t.Fatalf("expected E002 for missing comment block, got %v", diags)
	}
}

func TestParseTemplateNoPins(t *testing.T) {
	hlsl := "/* just a comment with no pins */"
	result, diags := ParseTemplate(hlsl)
	hasW002 := false
	for _, d := range diags {
		if d.Code == "W002" {
			hasW002 = true
		}
	}
	if !hasW002 {
		t.Error("expected W002 warning for 0 pins")
	}
	if len(result.Inputs) != 0 {
		t.Errorf("got %d inputs, want 0", len(result.Inputs))
	}
}

func TestParseTemplateDuplicatePin(t *testing.T) {
	hlsl := `/*
 * Pin 0 Name: [Foo] | Type suggestion: Scalar
 * Pin 1 Name: [Foo] | Type suggestion: Scalar
 */`
	result, diags := ParseTemplate(hlsl)
	hasE011 := false
	for _, d := range diags {
		if d.Code == "E011" {
			hasE011 = true
		}
	}
	if !hasE011 {
		t.Error("expected E011 for duplicate pin name")
	}
	if len(result.Inputs) != 1 {
		t.Errorf("got %d inputs, want 1 (deduped)", len(result.Inputs))
	}
}
