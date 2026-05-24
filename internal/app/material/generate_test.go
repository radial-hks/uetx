package material

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/radial/uetx/internal/domain"
	"github.com/radial/uetx/internal/material/build"
	"github.com/radial/uetx/internal/material/parser"
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

	expected := buildIRForTest(t, string(hlsl), "M_WaterLevel", 1)
	actual := buildIRForTest(t, string(hlsl), "M_WaterLevel", 2)
	assertGraphIsomorphism(t, expected, actual)
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
	resp := Inspect(domain.GenerateRequest{HLSL: string(hlsl)})
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

func TestInspect_WithOverrides(t *testing.T) {
	hlsl, err := os.ReadFile("../../../testdata/material/golden/M_WaterLevel.hlsl")
	if err != nil {
		t.Fatalf("read HLSL: %v", err)
	}
	resp := Inspect(domain.GenerateRequest{
		HLSL:         string(hlsl),
		MaterialName: "M_Test",
		OutputType:   domain.CMOTFloat3,
		Routing:      []string{"Emissive Color"},
	})
	if !resp.OK {
		t.Fatalf("inspect failed: %v", resp.Errors)
	}
	if resp.EffectiveOutputType != domain.CMOTFloat3 {
		t.Errorf("outputType = %q, want CMOT_Float3 (override)", resp.EffectiveOutputType)
	}
	if len(resp.EffectiveRouting) != 1 || resp.EffectiveRouting[0] != "Emissive Color" {
		t.Errorf("routing = %v, want [Emissive Color] (override)", resp.EffectiveRouting)
	}
	if resp.MaterialName != "M_Test" {
		t.Errorf("materialName = %q, want M_Test", resp.MaterialName)
	}
	// Parsed inputs should still come through when no explicit inputs given
	if len(resp.InferredInputs) != 7 {
		t.Errorf("inferred inputs = %d, want 7", len(resp.InferredInputs))
	}
}

func TestValidate_Basic(t *testing.T) {
	hlsl, err := os.ReadFile("../../../testdata/material/golden/M_WaterLevel.hlsl")
	if err != nil {
		t.Fatalf("read HLSL: %v", err)
	}
	resp := Validate(domain.GenerateRequest{HLSL: string(hlsl)})
	if !resp.OK {
		t.Fatalf("validate failed: %v", resp.Errors)
	}
}

func TestValidate_InvalidRouting(t *testing.T) {
	resp := Validate(domain.GenerateRequest{
		HLSL:    "/* Pin 0 Name: [X] | Type suggestion: Scalar */\nreturn X;",
		Routing: []string{"NotARealSlot"},
	})
	if resp.OK {
		t.Fatal("expected failure for invalid routing slot")
	}
	hasE102 := false
	for _, e := range resp.Errors {
		if e.Code == "E102" {
			hasE102 = true
		}
	}
	if !hasE102 {
		t.Error("expected E102 error")
	}
}

func TestValidate_InvalidOutputType(t *testing.T) {
	resp := Validate(domain.GenerateRequest{
		HLSL:       "/* Pin 0 Name: [X] | Type suggestion: Scalar */\nreturn X;",
		OutputType: "CMOT_Float9",
	})
	if resp.OK {
		t.Fatal("expected failure for invalid output type")
	}
	hasE101 := false
	for _, e := range resp.Errors {
		if e.Code == "E101" {
			hasE101 = true
		}
	}
	if !hasE101 {
		t.Error("expected E101 error")
	}
}

func buildIRForTest(t *testing.T, hlsl, matName string, seed int64) *build.BuildResult {
	t.Helper()
	parsed, _ := parser.ParseTemplate(hlsl)
	ot := parsed.OutputType
	if !parsed.HasOutputType {
		ot = domain.CMOTFloat3
	}
	result, diags := build.BuildIR(build.BuildRequest{
		HLSL:         hlsl,
		Inputs:       parsed.Inputs,
		OutputType:   ot,
		Routing:      domain.DefaultRouting(ot),
		MaterialName: matName,
	}, domain.NewSeededGUIDFunc(seed))
	for _, d := range diags {
		if d.Code[0] == 'E' {
			t.Fatalf("build error: %+v", d)
		}
	}
	return result
}

// assertGraphIsomorphism checks structural equivalence between two BuildResults
// (GUIDs differ but edges must match by GraphName+PinName, and Custom node Code must be non-empty).
func assertGraphIsomorphism(t *testing.T, expected, actual *build.BuildResult) {
	t.Helper()

	if len(expected.Nodes) != len(actual.Nodes) {
		t.Fatalf("node count: expected %d, got %d", len(expected.Nodes), len(actual.Nodes))
	}
	if len(expected.Edges) != len(actual.Edges) {
		t.Fatalf("edge count: expected %d, got %d", len(expected.Edges), len(actual.Edges))
	}

	expSet := edgeKeySet(t, expected)
	actSet := edgeKeySet(t, actual)
	for k := range expSet {
		if _, ok := actSet[k]; !ok {
			t.Errorf("missing edge in actual graph: %s", k)
		}
	}
	for k := range actSet {
		if _, ok := expSet[k]; !ok {
			t.Errorf("unexpected edge in actual graph: %s", k)
		}
	}

	assertCustomCodeNonEmpty(t, expected)
	assertCustomCodeNonEmpty(t, actual)
}

func edgeKeySet(t *testing.T, r *build.BuildResult) map[string]struct{} {
	t.Helper()
	pinName := make(map[string]string, len(r.Nodes)*8) // graphName|pinID -> pinName
	for _, n := range r.Nodes {
		for _, p := range n.Pins {
			pinName[n.GraphName+"|"+p.ID] = p.Name
		}
	}
	set := make(map[string]struct{}, len(r.Edges))
	for _, e := range r.Edges {
		fromName, ok := pinName[e.From.GraphName+"|"+e.From.PinID]
		if !ok {
			t.Fatalf("edge references unknown from pin: %s.%s", e.From.GraphName, e.From.PinID)
		}
		toName, ok := pinName[e.To.GraphName+"|"+e.To.PinID]
		if !ok {
			t.Fatalf("edge references unknown to pin: %s.%s", e.To.GraphName, e.To.PinID)
		}
		key := fmt.Sprintf("%s.%s -> %s.%s", e.From.GraphName, fromName, e.To.GraphName, toName)
		set[key] = struct{}{}
	}
	return set
}

func assertCustomCodeNonEmpty(t *testing.T, r *build.BuildResult) {
	t.Helper()
	for _, n := range r.Nodes {
		if n.ExprClass == "MaterialExpressionCustom" {
			if !strings.Contains(n.ExtraBody, "Code=\"") {
				t.Errorf("Custom node %s has no Code field", n.GraphName)
				return
			}
			// Code="..." — ensure inner content non-empty
			start := strings.Index(n.ExtraBody, "Code=\"")
			if start < 0 {
				t.Errorf("Custom node %s has no Code field", n.GraphName)
				return
			}
			rest := n.ExtraBody[start+len("Code=\""):]
			end := strings.Index(rest, "\"")
			if end <= 0 {
				t.Errorf("Custom node %s has empty Code", n.GraphName)
			}
			return
		}
	}
	t.Error("no Custom node found")
}

func TestValidate_InvalidMaterialName(t *testing.T) {
	resp := Validate(domain.GenerateRequest{
		HLSL:         "/* Pin 0 Name: [X] | Type suggestion: Scalar */\nreturn X;",
		MaterialName: "M Bad Name!",
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
