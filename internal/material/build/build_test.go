package build

import (
	"os"
	"testing"

	"github.com/radial/uetx/internal/domain"
	"github.com/radial/uetx/internal/material/parser"
)

func TestBuildIR_WaterLevel(t *testing.T) {
	hlsl, err := os.ReadFile("../../../testdata/material/golden/M_WaterLevel.hlsl")
	if err != nil {
		t.Fatalf("read HLSL: %v", err)
	}

	parsed, diags := parser.ParseTemplate(string(hlsl))
	for _, d := range diags {
		if d.Code[0] == 'E' {
			t.Fatalf("parse error: %s", d.Message)
		}
	}

	guidFn := domain.NewSeededGUIDFunc(42)
	result, bdiags := BuildIR(BuildRequest{
		HLSL:         string(hlsl),
		Inputs:       parsed.Inputs,
		OutputType:   parsed.OutputType,
		MaterialName: "M_WaterLevel",
	}, guidFn)

	for _, d := range bdiags {
		if d.Code[0] == 'E' {
			t.Fatalf("build error: %s", d.Message)
		}
	}

	// Expected: Root + 7 params + Custom + BreakOut = 10 nodes
	if got := len(result.Nodes); got != 10 {
		t.Errorf("node count = %d, want 10", got)
	}

	if !result.HasBreakOut {
		t.Error("expected BreakOut node (Float4 + Opacity is scalar)")
	}

	// Verify Root has 30 pins
	root := result.Nodes[0]
	if !root.IsRoot {
		t.Fatal("first node should be Root")
	}
	if got := len(root.Pins); got != 30 {
		t.Errorf("root pin count = %d, want 30", got)
	}

	// Find Custom node
	var custom *domain.GraphNode
	for _, n := range result.Nodes {
		if n.ExprClass == "MaterialExpressionCustom" {
			custom = n
			break
		}
	}
	if custom == nil {
		t.Fatal("no Custom node found")
	}
	// 7 input + 1 output = 8 pins
	if got := len(custom.Pins); got != 8 {
		t.Errorf("custom pin count = %d, want 8", got)
	}

	// Find BreakOut node
	var bo *domain.GraphNode
	for _, n := range result.Nodes {
		if n.ExprClass == "MaterialExpressionMaterialFunctionCall" {
			bo = n
			break
		}
	}
	if bo == nil {
		t.Fatal("no BreakOut node found")
	}
	// 1 input + 4 output = 5 pins
	if got := len(bo.Pins); got != 5 {
		t.Errorf("breakout pin count = %d, want 5", got)
	}

	// Verify edges: 7 (param->custom) + 1 (custom->breakout) + 2 (routing: custom->root.BaseColor + breakout.A->root.Opacity) = 10
	if got := len(result.Edges); got != 10 {
		t.Errorf("edge count = %d, want 10", got)
	}

	// Verify param node types
	paramTypes := map[string]int{}
	for _, n := range result.Nodes {
		if n.ExprClass != "" && n.ExprClass != "MaterialExpressionCustom" && n.ExprClass != "MaterialExpressionMaterialFunctionCall" {
			paramTypes[n.ExprClass]++
		}
	}
	if paramTypes["MaterialExpressionVectorParameter"] != 3 {
		t.Errorf("vector params = %d, want 3", paramTypes["MaterialExpressionVectorParameter"])
	}
	if paramTypes["MaterialExpressionScalarParameter"] != 4 {
		t.Errorf("scalar params = %d, want 4", paramTypes["MaterialExpressionScalarParameter"])
	}
}

func TestBuildIR_NoBreakOut(t *testing.T) {
	guidFn := domain.NewSeededGUIDFunc(99)
	result, _ := BuildIR(BuildRequest{
		HLSL:       "/* Pin 0 Name: [X] | Type suggestion: Scalar */\nreturn X;",
		Inputs:     []domain.NodeInput{{Name: "X", Type: domain.ParamScalar}},
		OutputType: domain.CMOTFloat3,
	}, guidFn)

	if result.HasBreakOut {
		t.Error("should not have BreakOut for Float3")
	}

	// Root + 1 param + Custom = 3 nodes
	if got := len(result.Nodes); got != 3 {
		t.Errorf("node count = %d, want 3", got)
	}
}

func TestBuildIR_Deterministic(t *testing.T) {
	hlsl := "/* Pin 0 Name: [A] | Type suggestion: Scalar */\nreturn A;"
	inputs := []domain.NodeInput{{Name: "A", Type: domain.ParamScalar}}

	fn1 := domain.NewSeededGUIDFunc(1)
	fn2 := domain.NewSeededGUIDFunc(1)

	r1, _ := BuildIR(BuildRequest{HLSL: hlsl, Inputs: inputs, OutputType: domain.CMOTFloat3}, fn1)
	r2, _ := BuildIR(BuildRequest{HLSL: hlsl, Inputs: inputs, OutputType: domain.CMOTFloat3}, fn2)

	if len(r1.Nodes) != len(r2.Nodes) {
		t.Fatal("node count mismatch")
	}
	for i := range r1.Nodes {
		if r1.Nodes[i].NodeGUID != r2.Nodes[i].NodeGUID {
			t.Errorf("node[%d] GUID mismatch", i)
		}
	}
}

func TestNeedsBreakOut(t *testing.T) {
	tests := []struct {
		ot      domain.OutputType
		routing []string
		want    bool
	}{
		{domain.CMOTFloat4, []string{"Opacity"}, true},
		{domain.CMOTFloat4, []string{"Base Color"}, false},
		{domain.CMOTFloat4, []string{"Base Color", "Opacity"}, true},
		{domain.CMOTFloat3, []string{"Opacity"}, false},
		{domain.CMOTFloat1, []string{"Metallic"}, false},
	}
	for _, tt := range tests {
		got := NeedsBreakOut(tt.ot, tt.routing)
		if got != tt.want {
			t.Errorf("NeedsBreakOut(%s, %v) = %v, want %v", tt.ot, tt.routing, got, tt.want)
		}
	}
}
